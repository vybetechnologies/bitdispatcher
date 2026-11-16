package mapdata

import "context"

// ExportSnapshot creates a consistent, standalone SQLite copy for an authorized
// offline MDT. VACUUM INTO excludes the live database's WAL and internal path.
func (s *Store) ExportSnapshot(ctx context.Context, path string) error {
	_, err := s.db.ExecContext(ctx, "VACUUM INTO ?", path)
	return err
}
