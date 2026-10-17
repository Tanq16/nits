package archive

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func IsEncrypted(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	magic := make([]byte, len(archiveMagic))
	_, err = io.ReadFull(f, magic)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return false, nil
		}
		return false, err
	}
	return bytes.Equal(magic, archiveMagic), nil
}

type ExtractPlan struct {
	Size        int64
	ArchiveSize int64
	reader      *zip.ReadCloser
	archive     string
	password    string
	tmp         string
	destDir     string
	bare        bool
}

func PrepareExtract(cfg ExtractConfig) (*ExtractPlan, error) {
	destDir, err := prepareDest(cfg.Dest)
	if err != nil {
		return nil, err
	}
	encrypted, err := IsEncrypted(cfg.Archive)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(cfg.Archive)
	if err != nil {
		return nil, err
	}
	plan := &ExtractPlan{ArchiveSize: info.Size(), archive: cfg.Archive, password: cfg.Password, destDir: destDir, bare: cfg.Bare}
	if encrypted {
		return plan, nil
	}
	return plan, plan.open(cfg.Archive)
}

func (p *ExtractPlan) Decrypt(progress io.Writer) error {
	tmp, err := decryptToTemp(p.archive, p.password, progress)
	if err != nil {
		return err
	}
	p.tmp = tmp
	if err := p.open(tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func (p *ExtractPlan) open(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	p.reader = r
	for _, f := range r.File {
		if !f.Mode().IsRegular() {
			continue
		}
		if _, ok := stripFirst(f.Name); p.bare && !ok {
			continue
		}
		p.Size += int64(f.UncompressedSize64)
	}
	return nil
}

func (p *ExtractPlan) Execute(progress io.Writer) error {
	defer func() {
		p.reader.Close()
		if p.tmp != "" {
			os.Remove(p.tmp)
		}
	}()
	for _, f := range p.reader.File {
		if err := extractOne(f, p.destDir, p.bare, progress); err != nil {
			return err
		}
	}
	return nil
}

func decryptToTemp(path, password string, progress io.Writer) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password required")
	}
	src, err := os.Open(path)
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp("", "nits-archive-*.zip")
	if err != nil {
		src.Close()
		return "", err
	}
	if err := errors.Join(decryptTo(tmp, io.TeeReader(src, progress), password), src.Close(), tmp.Close()); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

func extractOne(f *zip.File, destDir string, bare bool, progress io.Writer) error {
	mode := f.Mode()
	if mode&os.ModeSymlink != 0 {
		return nil
	}
	target, ok, err := entryTarget(destDir, f.Name, bare)
	if !ok {
		return err
	}
	info := f.FileInfo()
	if info.IsDir() || strings.HasSuffix(f.Name, "/") {
		return os.MkdirAll(target, 0755)
	}
	if !info.Mode().IsRegular() && !mode.IsRegular() {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	return errors.Join(writeEntry(target, info.Mode().Perm(), io.TeeReader(rc, progress)), rc.Close())
}

func prepareDest(dest string) (string, error) {
	destDir, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	destDir = filepath.Clean(destDir)
	return destDir, os.MkdirAll(destDir, 0755)
}

func entryTarget(destDir, name string, bare bool) (string, bool, error) {
	if bare {
		stripped, ok := stripFirst(name)
		if !ok {
			return "", false, nil
		}
		name = stripped
	}
	target, err := safeExtractPath(destDir, name)
	if err != nil {
		return "", false, err
	}
	return target, true, nil
}

func writeEntry(target string, perm os.FileMode, src io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	if perm == 0 {
		perm = 0644
	}
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, src)
	return errors.Join(err, out.Close())
}

func stripFirst(name string) (string, bool) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "/")
	slash := strings.IndexByte(name, '/')
	if slash < 0 {
		return "", false
	}
	rest := name[slash+1:]
	if rest == "" {
		return "", false
	}
	return rest, true
}

func safeExtractPath(destDir, name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	target := filepath.Join(destDir, filepath.FromSlash(name))
	target = filepath.Clean(target)
	sep := string(os.PathSeparator)
	if target != destDir && !strings.HasPrefix(target, destDir+sep) {
		return "", fmt.Errorf("illegal path in archive: %s", name)
	}
	return target, nil
}
