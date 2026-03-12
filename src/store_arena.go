package main

import (
	"encoding/json"
	"strings"
)

func (s *Store) listLatestAgentFillsForAssignedAgents() (map[string]AgentArenaLatestFill, error) {
	rows, err := s.db.Query(`
		SELECT f.id, f.public_key, f.fill_time, f.data_json
		FROM agent_fills f
		INNER JOIN agent_accounts a ON lower(a.public_key) = lower(f.public_key)
		WHERE a.status = 'assigned'
		  AND f.id = (
			SELECT f2.id
			FROM agent_fills f2
			WHERE lower(f2.public_key) = lower(f.public_key)
			ORDER BY f2.fill_time DESC, f2.created_at DESC, f2.id DESC
			LIMIT 1
		  )`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make(map[string]AgentArenaLatestFill)
	for rows.Next() {
		var item AgentArenaLatestFill
		var raw string
		if err := rows.Scan(&item.ID, &item.PublicKey, &item.FillTime, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &item.Fill); err != nil {
			continue
		}
		item.PublicKey = strings.ToLower(strings.TrimSpace(item.PublicKey))
		items[item.PublicKey] = item
	}
	return items, rows.Err()
}
