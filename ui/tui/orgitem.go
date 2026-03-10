package tui

import "github.com/iamoeg/bootdev-capstone/internal/domain"

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
	return desc
}

func (i orgItem) FilterValue() string {
	return i.org.Name + " " + i.org.Activity + " " + i.org.ICENum
}
