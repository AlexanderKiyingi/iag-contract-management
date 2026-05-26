package models

import "strings"

const maxProfilePhotoBytes = 2 * 1024 * 1024

func (s *Store) GetProfilePhoto(email string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Frontend.ProfilePhotos == nil {
		return ""
	}
	return s.Frontend.ProfilePhotos[strings.ToLower(strings.TrimSpace(email))]
}

func (s *Store) ValidateProfilePhotoDataURL(dataURL string) error {
	if dataURL == "" {
		return nil
	}
	if len(dataURL) > maxProfilePhotoBytes {
		return ErrValidation
	}
	if !strings.HasPrefix(dataURL, "data:image/") {
		return ErrValidation
	}
	return nil
}
