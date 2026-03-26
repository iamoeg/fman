package tui

import (
	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"time"
)

// orgItem wraps a domain.Organization to satisfy list.DefaultItem.
type orgItem struct {
	org *domain.Organization
}

func (i orgItem) Title() string {
	return i.org.Name
}

func (i orgItem) Description() string {
	desc := string(i.org.LegalForm)
	if i.org.Activity != "" {
		desc += "  ·  " + i.org.Activity
	}
	if i.org.ICENum != "" {
		desc += "  ·  ICE: " + i.org.ICENum
	}
	if i.org.DeletedAt != nil {
		desc += "  ·  deleted: " + i.org.DeletedAt.Format(time.DateOnly)
	}
	return desc
}

func (i orgItem) FilterValue() string {
	return i.org.Name + " " + i.org.Activity + " " + i.org.ICENum
}
