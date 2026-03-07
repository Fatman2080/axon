package main

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type BackupInfo struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"createdAt"`
}

func (s *Server) backupDir() string {
	return filepath.Join(filepath.Dir(s.dbPath), "backups")
}

func (s *Server) performBackup() (BackupInfo, error) {
	if !atomic.CompareAndSwapInt32(&s.backupRunning, 0, 1) {
		return BackupInfo{}, fmt.Errorf("backup already in progress")
	}
	defer atomic.StoreInt32(&s.backupRunning, 0)

	dir := s.backupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return BackupInfo{}, fmt.Errorf("create backup dir: %w", err)
	}

	ts := time.Now().UTC().Format("20060102_150405")
	finalName := fmt.Sprintf("openfi_backup_%s.db.gz", ts)
	finalPath := filepath.Join(dir, finalName)

	// VACUUM INTO a temp .db file (consistent snapshot, no exclusive lock)
	tmpDB := filepath.Join(dir, fmt.Sprintf("openfi_backup_%s.db", ts))
	_, err := s.store.db.Exec(fmt.Sprintf(`VACUUM INTO '%s'`, tmpDB))
	if err != nil {
		return BackupInfo{}, fmt.Errorf("vacuum into: %w", err)
	}

	// Gzip compress
	tmpGz := finalPath + ".tmp"
	if err := gzipFile(tmpDB, tmpGz); err != nil {
		os.Remove(tmpDB)
		os.Remove(tmpGz)
		return BackupInfo{}, fmt.Errorf("gzip: %w", err)
	}
	os.Remove(tmpDB)

	if err := os.Rename(tmpGz, finalPath); err != nil {
		os.Remove(tmpGz)
		return BackupInfo{}, fmt.Errorf("rename: %w", err)
	}

	s.cleanupOldBackups()

	_ = s.store.setSetting("backup_last_at", nowISO())

	fi, _ := os.Stat(finalPath)
	info := BackupInfo{
		Name:      finalName,
		Size:      fi.Size(),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	logInfo("backup", "created %s (%.1f KB)", finalName, float64(fi.Size())/1024)
	return info, nil
}

func gzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	gz, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := io.Copy(gz, in); err != nil {
		gz.Close()
		return err
	}
	return gz.Close()
}

func (s *Server) listBackups() ([]BackupInfo, error) {
	dir := s.backupDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "openfi_backup_") || !strings.HasSuffix(e.Name(), ".db.gz") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		createdAt := parseBackupTimestamp(e.Name())
		backups = append(backups, BackupInfo{
			Name:      e.Name(),
			Size:      fi.Size(),
			CreatedAt: createdAt,
		})
	}

	// Sort newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt > backups[j].CreatedAt
	})

	return backups, nil
}

func parseBackupTimestamp(name string) string {
	// openfi_backup_20060102_150405.db.gz
	name = strings.TrimPrefix(name, "openfi_backup_")
	name = strings.TrimSuffix(name, ".db.gz")
	t, err := time.Parse("20060102_150405", name)
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// cleanupOldBackups applies a GFS (Grandfather-Father-Son) tiered retention:
//
//   - Hourly tier: keep the N most recent backups (by count, regardless of time).
//   - Daily  tier: keep the newest backup from each of the M most recent distinct
//     calendar days (UTC). This adds coverage beyond the hourly window.
//   - Weekly tier: keep the newest backup from each of the K most recent distinct
//     ISO weeks. This adds long-term coverage.
//
// A backup retained by ANY tier survives; all others are deleted.
func (s *Server) cleanupOldBackups() {
	retainHourly := getPositiveIntSetting(s.store, "backup_retain_hourly", 3)
	retainDaily := getPositiveIntSetting(s.store, "backup_retain_daily", 3)
	retainWeekly := getPositiveIntSetting(s.store, "backup_retain_weekly", 3)

	backups, err := s.listBackups() // sorted newest-first
	if err != nil || len(backups) == 0 {
		return
	}

	keep := make(map[string]bool)

	// --- hourly tier: simply keep the N most recent backups ---
	for i, b := range backups {
		if i >= retainHourly {
			break
		}
		keep[b.Name] = true
	}

	// --- daily tier: 1 per calendar day, M most recent distinct days ---
	dayBuckets := make(map[string]bool)
	kept := 0
	for _, b := range backups {
		if kept >= retainDaily {
			break
		}
		t := parseBackupTime(b.Name)
		if t.IsZero() {
			continue
		}
		bucket := t.Format("2006-01-02")
		if dayBuckets[bucket] {
			continue
		}
		dayBuckets[bucket] = true
		keep[b.Name] = true
		kept++
	}

	// --- weekly tier: 1 per ISO week, K most recent distinct weeks ---
	weekBuckets := make(map[string]bool)
	kept = 0
	for _, b := range backups {
		if kept >= retainWeekly {
			break
		}
		t := parseBackupTime(b.Name)
		if t.IsZero() {
			continue
		}
		y, w := t.ISOWeek()
		bucket := fmt.Sprintf("%d-W%02d", y, w)
		if weekBuckets[bucket] {
			continue
		}
		weekBuckets[bucket] = true
		keep[b.Name] = true
		kept++
	}

	// Delete anything not retained by any tier
	dir := s.backupDir()
	removed := 0
	for _, b := range backups {
		if keep[b.Name] {
			continue
		}
		path := filepath.Join(dir, b.Name)
		if err := os.Remove(path); err != nil {
			logWarn("backup", "failed to remove old backup %s: %v", b.Name, err)
		} else {
			removed++
		}
	}
	if removed > 0 {
		logInfo("backup", "retention cleanup: removed %d backups (keep hourly=%d daily=%d weekly=%d)",
			removed, retainHourly, retainDaily, retainWeekly)
	}
}

// parseBackupTime returns the UTC time encoded in a backup filename, or zero.
func parseBackupTime(name string) time.Time {
	s := strings.TrimPrefix(name, "openfi_backup_")
	s = strings.TrimSuffix(s, ".db.gz")
	t, err := time.Parse("20060102_150405", s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func getPositiveIntSetting(store *Store, key string, fallback int) int {
	v, err := strconv.Atoi(store.getSettingDefault(key, strconv.Itoa(fallback)))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func (s *Server) restoreFromBackup(name string) error {
	dir := s.backupDir()
	backupPath := filepath.Join(dir, name)

	// Validate name to prevent path traversal
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid backup name")
	}
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %s", name)
	}

	// Stop sync goroutine
	if s.syncStop != nil {
		close(s.syncStop)
		s.syncStop = nil
	}
	// Stop backup goroutine
	if s.backupStop != nil {
		close(s.backupStop)
		s.backupStop = nil
	}

	// Decompress to temp file
	tmpDB := filepath.Join(dir, "restore_tmp.db")
	if err := gunzipFile(backupPath, tmpDB); err != nil {
		s.restartGoroutines()
		return fmt.Errorf("decompress backup: %w", err)
	}

	// Validate with integrity check
	checkDB, err := sql.Open("sqlite3", tmpDB)
	if err != nil {
		os.Remove(tmpDB)
		s.restartGoroutines()
		return fmt.Errorf("open backup db: %w", err)
	}
	var result string
	if err := checkDB.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil || result != "ok" {
		checkDB.Close()
		os.Remove(tmpDB)
		s.restartGoroutines()
		return fmt.Errorf("backup integrity check failed: %s", result)
	}
	checkDB.Close()

	// Create pre-restore safety backup of current DB
	safetyName := fmt.Sprintf("openfi_prerestore_%s.db.gz", time.Now().UTC().Format("20060102_150405"))
	safetyPath := filepath.Join(dir, safetyName)
	safetyTmpDB := filepath.Join(dir, "safety_tmp.db")
	if _, err := s.store.db.Exec(fmt.Sprintf(`VACUUM INTO '%s'`, safetyTmpDB)); err == nil {
		if err := gzipFile(safetyTmpDB, safetyPath); err == nil {
			logInfo("backup", "pre-restore safety backup: %s", safetyName)
		}
		os.Remove(safetyTmpDB)
	}

	// Close current DB
	s.store.db.Close()

	// Remove WAL/SHM files
	os.Remove(s.dbPath + "-wal")
	os.Remove(s.dbPath + "-shm")

	// Replace DB file
	if err := os.Rename(tmpDB, s.dbPath); err != nil {
		os.Remove(tmpDB)
		s.restartGoroutines()
		return fmt.Errorf("replace db: %w", err)
	}

	// Re-open DB
	newDB, err := sql.Open("sqlite3", s.dbPath)
	if err != nil {
		s.restartGoroutines()
		return fmt.Errorf("reopen db: %w", err)
	}
	s.store.db = newDB

	// Apply any newer migrations
	if err := s.store.initSchema(); err != nil {
		logError("backup", "post-restore initSchema: %v", err)
	}

	// Reload all in-memory settings from the restored DB
	s.reloadSettingsFromDB()

	// Restart goroutines (reads interval from DB)
	s.restartGoroutines()

	// Refresh public cache
	s.refreshPublicCache()

	logInfo("backup", "restored from %s", name)
	return nil
}

// reloadSettingsFromDB re-reads every DB-sourced setting into the Server's
// in-memory fields so the running process matches the restored database.
func (s *Server) reloadSettingsFromDB() {
	// --- X OAuth ---
	s.xOAuth.ClientID = s.store.getSettingDefault("xoauth_client_id", "")
	s.xOAuth.ClientSecret = s.store.getSettingDefault("xoauth_client_secret", "")
	s.xOAuth.Scopes = s.store.getSettingDefault("xoauth_scopes", "users.read tweet.read offline.access")
	logInfo("backup", "reloaded xOAuth settings (clientId: %s...)", truncate(s.xOAuth.ClientID, 8))

	// --- Contracts + EVM client ---
	newRPC := s.store.getSettingDefault("contracts_rpc_url", "")
	newAllocator := s.store.getSettingDefault("contracts_allocator_address", "")
	rpcChanged := newRPC != s.contractRPCURL || newAllocator != s.contractAllocator
	s.contractRPCURL = newRPC
	s.contractAllocator = newAllocator

	if rpcChanged && newRPC != "" && newAllocator != "" {
		go func() {
			logInfo("evm", "reinitializing EVM client after restore (rpc: %s)", newRPC)
			ec, err := NewEVMClient(newRPC, newAllocator)
			if err != nil {
				logWarn("evm", "EVM client reinit failed: %v", err)
				return
			}
			s.setEVMClient(ec)
			logInfo("evm", "EVM client ready")
		}()
	} else if newRPC == "" || newAllocator == "" {
		s.setEVMClient(nil)
	}

	logInfo("backup", "reloaded contracts settings (rpc: %s, allocator: %s)", truncate(newRPC, 20), truncate(newAllocator, 12))
}

func (s *Server) restartGoroutines() {
	// Restart sync (reads interval from restored DB)
	syncIntervalStr := s.store.getSettingDefault("sync_interval_seconds", "60")
	syncInterval, _ := strconv.Atoi(syncIntervalStr)
	s.startAutoSync(syncInterval)

	// Restart backup
	s.startAutoBackup()
}

func gunzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	gz, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gz.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, gz)
	return err
}

func (s *Server) startAutoBackup() {
	if s.backupStop != nil {
		close(s.backupStop)
		s.backupStop = nil
	}

	stop := make(chan struct{})
	s.backupStop = stop

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.checkAndRunBackup()
			case <-stop:
				return
			}
		}
	}()
	logInfo("backup", "auto backup goroutine started")
}

func (s *Server) checkAndRunBackup() {
	intervalStr := s.store.getSettingDefault("backup_interval_hours", "24")
	intervalHours, err := strconv.Atoi(intervalStr)
	if err != nil || intervalHours <= 0 {
		return // disabled
	}

	lastAtStr := s.store.getSettingDefault("backup_last_at", "")
	if lastAtStr != "" {
		lastAt, err := time.Parse(time.RFC3339, lastAtStr)
		if err == nil {
			nextDue := lastAt.Add(time.Duration(intervalHours) * time.Hour)
			if time.Now().UTC().Before(nextDue) {
				return // not due yet
			}
		}
	}

	logInfo("backup", "auto backup triggered (interval: %dh)", intervalHours)
	if _, err := s.performBackup(); err != nil {
		logError("backup", "auto backup failed: %v", err)
	}
}
