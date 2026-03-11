package tui

import (
	"fmt"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

type compItem struct {
	pkg *domain.EmployeeCompensationPackage
}

func (i compItem) Title() string {
	return i.pkg.Name
}

func (i compItem) Description() string {
	return fmt.Sprintf("%.2f %s · Created %s", i.pkg.BaseSalary.ToMAD(), i.pkg.Currency, i.pkg.CreatedAt.Format("2006-01-02"))
}

func (i compItem) FilterValue() string {
	return i.pkg.Name
}
