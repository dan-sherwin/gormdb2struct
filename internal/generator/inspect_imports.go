package generator

import (
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type inspectionImportedPackage struct {
	ImportPath    string
	PackageName   string
	ExportedTypes map[string]struct{}
}

var inspectionBuiltinImportPaths = map[string]struct{}{
	"github.com/dan-sherwin/gormdb2struct/pgtypes": {},
	"gorm.io/datatypes":                            {},
}

const inspectionCurrentModulePath = "github.com/dan-sherwin/gormdb2struct"

func loadInspectionImportedPackages(ctx context.Context, importPaths []string) ([]inspectionImportedPackage, error) {
	candidates := filterInspectionImportPaths(importPaths)
	if len(candidates) == 0 {
		return nil, nil
	}

	localCandidates, externalCandidates := splitInspectionImportPaths(candidates)

	imported := make([]inspectionImportedPackage, 0, len(candidates))
	if len(localCandidates) > 0 {
		loaded, err := loadInspectionPackages(ctx, "", localCandidates)
		if err != nil {
			return nil, err
		}
		imported = append(imported, loaded...)
	}

	if len(externalCandidates) > 0 {
		loaded, err := loadInspectionPackagesInTempModule(ctx, externalCandidates)
		if err != nil {
			return nil, err
		}
		imported = append(imported, loaded...)
	}

	sort.Slice(imported, func(i, j int) bool { return imported[i].ImportPath < imported[j].ImportPath })
	return imported, nil
}

func loadInspectionPackages(ctx context.Context, dir string, importPaths []string) ([]inspectionImportedPackage, error) {
	loaded, err := packages.Load(&packages.Config{
		Context: ctx,
		Mode:    packages.NeedName | packages.NeedTypes,
		Dir:     dir,
		Env:     append(os.Environ(), "GOWORK=off"),
	}, importPaths...)
	if err != nil {
		return nil, fmt.Errorf("load import packages: %w", err)
	}

	imported := make([]inspectionImportedPackage, 0, len(loaded))
	for _, pkg := range loaded {
		if pkg == nil {
			continue
		}
		if len(pkg.Errors) > 0 {
			return nil, fmt.Errorf("load import package %q: %s", pkg.PkgPath, pkg.Errors[0].Msg)
		}
		if pkg.Types == nil {
			return nil, fmt.Errorf("load import package %q: missing type information", pkg.PkgPath)
		}

		exportedTypes := make(map[string]struct{})
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			if !ast.IsExported(name) {
				continue
			}
			if _, ok := scope.Lookup(name).(*types.TypeName); !ok {
				continue
			}
			exportedTypes[name] = struct{}{}
		}

		imported = append(imported, inspectionImportedPackage{
			ImportPath:    pkg.PkgPath,
			PackageName:   pkg.Name,
			ExportedTypes: exportedTypes,
		})
	}

	return imported, nil
}

func loadInspectionPackagesInTempModule(ctx context.Context, importPaths []string) ([]inspectionImportedPackage, error) {
	tempDir, err := os.MkdirTemp("", "gormdb2struct-imports-*")
	if err != nil {
		return nil, fmt.Errorf("create temp module dir for import packages: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module gormdb2struct-imports-temp\n\ngo 1.26.2\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write temp module go.mod: %w", err)
	}

	for _, importPath := range importPaths {
		cmd := exec.CommandContext(ctx, "go", "get", importPath)
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		output, err := cmd.CombinedOutput()
		if err != nil {
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = err.Error()
			}
			return nil, fmt.Errorf("load import package %q: %s", importPath, message)
		}
	}

	return loadInspectionPackages(ctx, tempDir, importPaths)
}

func filterInspectionImportPaths(importPaths []string) []string {
	seen := make(map[string]struct{}, len(importPaths))
	out := make([]string, 0, len(importPaths))
	for _, importPath := range importPaths {
		cleaned := strings.TrimSpace(importPath)
		if cleaned == "" {
			continue
		}
		if _, builtin := inspectionBuiltinImportPaths[cleaned]; builtin {
			continue
		}
		if _, exists := seen[cleaned]; exists {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}
	sort.Strings(out)
	return out
}

func splitInspectionImportPaths(importPaths []string) (local []string, external []string) {
	for _, importPath := range importPaths {
		if importPath == inspectionCurrentModulePath || strings.HasPrefix(importPath, inspectionCurrentModulePath+"/") {
			local = append(local, importPath)
			continue
		}
		external = append(external, importPath)
	}
	return local, external
}

func applyInspectionImportRecommendations(report *InspectionReport, imported []inspectionImportedPackage) {
	if report == nil || len(report.Findings) == 0 || len(imported) == 0 {
		return
	}

	index := make(map[string][]inspectionImportedPackage)
	for _, pkg := range imported {
		for typeName := range pkg.ExportedTypes {
			index[typeName] = append(index[typeName], pkg)
		}
	}

	for i := range report.Findings {
		finding := &report.Findings[i]
		if finding.CurrentMapping != "" {
			continue
		}
		if finding.Recommendation != InspectionRecommendationGenerate && finding.Recommendation != InspectionRecommendationManual {
			continue
		}

		typeName, ok := inspectionSuggestedTypeName(*finding)
		if !ok {
			continue
		}
		matches := index[typeName]
		if len(matches) != 1 {
			continue
		}

		finding.Recommendation = InspectionRecommendationTypeMap
		finding.SuggestedImportPath = matches[0].ImportPath
		finding.SuggestedImportPackage = matches[0].PackageName
		finding.SuggestedGoType = matches[0].PackageName + "." + typeName
		if strings.Contains(finding.Note, "auto-generated") || strings.Contains(finding.Note, "generated in PostgreSQL.GeneratedTypes.TypeMap") {
			finding.Note = ""
		}
	}
}

func inspectionSuggestedTypeName(finding InspectionTypeFinding) (string, bool) {
	cleaned := strings.TrimSpace(finding.SuggestedGoType)
	if cleaned == "" || strings.Contains(cleaned, ".") {
		return "", false
	}
	if !ast.IsExported(cleaned) {
		return "", false
	}
	return cleaned, true
}
