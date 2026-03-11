package tui

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

type empItem struct {
	emp     *domain.Employee
	pkgName string // resolved compensation package name
}

func (i empItem) Title() string {
	return fmt.Sprintf("#%d · %s", i.emp.SerialNum, i.emp.FullName)
}

func (i empItem) Description() string {
	return fmt.Sprintf("%s · %s", i.emp.Position, i.pkgName)
}

func (i empItem) FilterValue() string {
	return i.emp.FullName
}

// pkgNameMap builds a uuid→name lookup from a package slice.
func pkgNameMap(pkgs []*domain.EmployeeCompensationPackage) map[uuid.UUID]string {
	m := make(map[uuid.UUID]string, len(pkgs))
	for _, p := range pkgs {
		m[p.ID] = p.Name
	}
	return m
}
