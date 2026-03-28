package unity

import (
	"os"
	"path/filepath"
	"strings"
)

// WriteCscRsp writes a csc.rsp file with -nowarn directives for the given warning codes.
// The file is placed at the given path (typically next to an .asmdef file).
func WriteCscRsp(dir string, warnings []string) error {
	if len(warnings) == 0 {
		return nil
	}
	content := "-nowarn:" + strings.Join(warnings, ",") + "\n"
	return os.WriteFile(filepath.Join(dir, "csc.rsp"), []byte(content), 0644)
}

// WriteCscRspForAsmdefs finds all .asmdef files under rootDir and writes a csc.rsp
// next to each one. This is used for packages that already contain their own .asmdef
// files (git-unity and archive-unity).
func WriteCscRspForAsmdefs(rootDir string, warnings []string) error {
	if len(warnings) == 0 {
		return nil
	}
	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".asmdef") {
			dir := filepath.Dir(path)
			if err := WriteCscRsp(dir, warnings); err != nil {
				return err
			}
		}
		return nil
	})
}
