package go_mod

import (
	"context"
	"github.com/murphysecurity/murphysec/model"
	"github.com/murphysecurity/murphysec/utils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/mod/modfile"
	"path/filepath"
)

type Inspector struct{}

func (i *Inspector) SupportFeature(feature model.InspectorFeature) bool {
	return model.InspectorFeatureAllowNested&feature > 0
}

func (i *Inspector) String() string {
	return "GoMod"
}

func (i *Inspector) CheckDir(dir string) bool {
	return utils.IsFile(filepath.Join(dir, "go.mod"))
}

func (i *Inspector) InspectProject(ctx context.Context) error {
	task := model.UseInspectorTask(ctx)
	logger := utils.UseLogger(ctx)
	modFilePath := filepath.Join(task.ScanDir, "go.mod")
	logger.Debug("Reading go.mod", zap.String("path", modFilePath))
	data, e := utils.ReadFileLimited(modFilePath, 1024*1024*4)
	if e != nil {
		return errors.WithMessage(e, "Open GoMod file")
	}
	logger.Debug("Parsing go.mod")
	f, e := modfile.Parse(filepath.Base(modFilePath), data, nil)
	if e != nil {
		return errors.WithMessage(e, "Parse go mod failed")
	}
	m := model.Module{
		PackageManager: model.PMGoMod,
		Language:       model.Go,
		RelativePath:   modFilePath,
		Name:           "<NoNameModule>",
	}
	if f.Module != nil {
		m.Version = f.Module.Mod.Version
		m.Name = f.Module.Mod.Path
	}

	var depM = make(map[[2]string]struct{})
	for _, it := range f.Require {
		if it == nil {
			continue
		}
		depM[[2]string{it.Mod.Path, it.Mod.Version}] = struct{}{}
	}
	for _, it := range f.Replace {
		delete(depM, [2]string{it.Old.Path, it.Old.Version})
		depM[[2]string{it.New.Path, it.New.Version}] = struct{}{}
	}
	for it := range depM {
		m.Dependencies = append(m.Dependencies, model.Dependency{
			Name:    it[0],
			Version: it[1],
		})
	}
	task.AddModule(m)
	return nil
}
