package main

func (s *Store) seedIfEmpty() error {
	if err := s.seedAdminIfNeeded(); err != nil {
		return err
	}
	return nil
}

func (s *Store) seedAdminIfNeeded() error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM admins`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	tempPassword, err := randomURLSafeString(12)
	if err != nil {
		return err
	}
	const defaultEmail = "admin@openfi.local"
	const defaultName = "System Admin"
	passwordHash, err := hashPasswordStrong(tempPassword)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO admins(id, email, password_hash, name, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, newID("admin"), defaultEmail, passwordHash, defaultName, nowISO())
	if err == nil {
		logInfo("seed", "bootstrap admin created: email=%s temporary_password=%s (please change immediately)", defaultEmail, tempPassword)
	}
	return err
}
