package tui

import (
	"fmt"
	"time"

	"github.com/iamoeg/fman/internal/domain"
)

type compItem struct {
	pkg *domain.EmployeeCompensationPackage
}

func (i compItem) Title() string {
	return i.pkg.Name
}

func (i compItem) Description() string {
	desc := fmt.Sprintf("%.2f %s · Created %s", i.pkg.BaseSalary.ToMAD(), i.pkg.Currency, i.pkg.CreatedAt.Format("2006-01-02"))
	if i.pkg.DeletedAt != nil {
		desc += "  ·  deleted: " + i.pkg.DeletedAt.Format(time.DateOnly)
	}
	return desc
}

func (i compItem) FilterValue() string {
	return i.pkg.Name
}
