package analyzer

import (
	"os"
	"path/filepath"

	"github.com/knqyf263/fanal/extractor"
	"github.com/pkg/errors"
)

var (
	osAnalyzers  []OSAnalyzer
	pkgAnalyzers []PkgAnalyzer

	// ErrUnknownOS occurs when unknown OS is analyzed.
	ErrUnknownOS = errors.New("Unknown OS")
	// ErrPkgAnalysis occurs when the analysis of packages is failed.
	ErrPkgAnalysis = errors.New("Failed to analyze packages")
)

type OSAnalyzer interface {
	Analyze(extractor.FilesMap) (OS, error)
	RequiredFiles() []string
}

type PkgAnalyzer interface {
	Analyze(extractor.FilesMap) ([]Package, error)
	RequiredFiles() []string
}

type OS struct {
	Name   string
	Family string
}

type Package struct {
	Name    string
	Version string
	Release string
	Epoch   int
}

func RegisterOSAnalyzer(analyzer OSAnalyzer) {
	osAnalyzers = append(osAnalyzers, analyzer)
}

func RegisterPkgAnalyzer(analyzer PkgAnalyzer) {
	pkgAnalyzers = append(pkgAnalyzers, analyzer)
}

func RequiredFilenames() []string {
	filenames := []string{}
	for _, analyzer := range osAnalyzers {
		filenames = append(filenames, analyzer.RequiredFiles()...)
	}
	for _, analyzer := range pkgAnalyzers {
		filenames = append(filenames, analyzer.RequiredFiles()...)
	}
	return filenames
}

func Analyze(dir string) (filesMap extractor.FilesMap, err error) {
	extractor := extractor.DockerExtractor{}
	file, _ := os.Open(filepath.Join(dir, "layer.tar"))

	filesMap, err = extractor.ExtractFiles(file, RequiredFilenames())
	if err != nil {
		return nil, errors.Wrap(err, "Failed to extract files")
	}
	return filesMap, nil
}

func GetOS(filesMap extractor.FilesMap) (OS, error) {
	for _, analyzer := range osAnalyzers {
		os, err := analyzer.Analyze(filesMap)
		if err != nil {
			continue
		}
		return os, nil
	}
	return OS{}, ErrUnknownOS

}

func GetPackages(filesMap extractor.FilesMap) ([]Package, error) {
	for _, analyzer := range pkgAnalyzers {
		pkgs, err := analyzer.Analyze(filesMap)
		if err != nil {
			continue
		}
		return pkgs, nil
	}
	return nil, ErrUnknownOS

}
