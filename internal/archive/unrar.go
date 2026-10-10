package archive

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/nwaples/rardecode/v2"
)

func IsRarEncrypted(path string) (bool, error) {
	rc, err := rardecode.OpenReader(path)
	if errors.Is(err, rardecode.ErrArchiveEncrypted) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer rc.Close()
	for {
		hdr, err := rc.Next()
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		if errors.Is(err, rardecode.ErrArchiveEncrypted) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		if hdr.Encrypted {
			return true, nil
		}
	}
}

func Unrar(cfg ExtractConfig) error {
	destDir, err := prepareDest(cfg.Dest)
	if err != nil {
		return err
	}
	var opts []rardecode.Option
	if cfg.Password != "" {
		opts = append(opts, rardecode.Password(cfg.Password))
	}
	rc, err := rardecode.OpenReader(cfg.Archive, opts...)
	if err != nil {
		return err
	}
	defer rc.Close()
	for {
		hdr, err := rc.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := unrarOne(rc, hdr, destDir, cfg.Bare); err != nil {
			return err
		}
	}
}

func unrarOne(r io.Reader, hdr *rardecode.FileHeader, destDir string, bare bool) error {
	if hdr.Mode()&os.ModeSymlink != 0 || hdr.LinkType == rardecode.LinkTypeWindowsJunction {
		return nil
	}
	target, ok, err := entryTarget(destDir, hdr.Name, bare)
	if !ok {
		return err
	}
	if hdr.IsDir {
		return os.MkdirAll(target, 0755)
	}
	perm := hdr.Mode().Perm()
	if hdr.LinkType != rardecode.LinkTypeHardLink && hdr.LinkType != rardecode.LinkTypeFileCopy {
		return writeEntry(target, perm, r)
	}
	source, ok, err := entryTarget(destDir, hdr.LinkTarget, bare)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("link target not extracted: %s", hdr.LinkTarget)
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	return errors.Join(writeEntry(target, perm, src), src.Close())
}
