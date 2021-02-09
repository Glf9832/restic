package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"restic/storage"
)

func restore_file(repo *storage.DirRepository, node storage.Node, target string) error {
	fmt.Printf("  restore file %q\n", target)

	rd, err := repo.Get(node.Content)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, 0600)
	defer f.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(f, rd)
	if err != nil {
		return err
	}

	err = f.Chmod(node.Mode)
	if err != nil {
		return err
	}

	err = f.Chown(int(node.User), int(node.Group))
	if err != nil {
		return err
	}

	err = os.Chtimes(target, node.AccessTime, node.ModTime)
	if err != nil {
		return err
	}

	return nil
}

func restore_dir(repo *storage.DirRepository, id storage.ID, target string) error {
	fmt.Printf("  restore dir %q\n", target)
	rd, err := repo.Get(id)
	if err != nil {
		return err
	}

	t := storage.NewTree()
	err = t.Restore(rd)
	if err != nil {
		return err
	}

	for _, node := range t.Nodes {
		name := path.Base(node.Name)
		if name == "." || name == ".." {
			return errors.New("invalid path")
		}

		nodepath := path.Join(target, name)
		if node.Mode.IsDir() {
			err = os.MkdirAll(nodepath, 0700)
			if err != nil {
				return err
			}

			err = os.Chmod(nodepath, node.Mode)
			if err != nil {
				return err
			}

			err = os.Chown(nodepath, int(node.User), int(node.Group))
			if err != nil {
				return err
			}

			err = os.Chtimes(nodepath, node.AccessTime, node.ModTime)
			if err != nil {
				return err
			}

			err = restore_dir(repo, node.Content, nodepath)

			if err != nil {
				return err
			}
		} else {
			err = restore_file(repo, node, nodepath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func commandRestore(repo *storage.DirRepository, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: restore ID dir")
	}

	id, err := storage.ParseID(args[0])
	if err != nil {
		errmsg(1, "invalid id %q: %v", args[0], err)
	}

	target := args[1]

	err = os.MkdirAll(target, 0700)
	if err != nil {
		return err
	}

	err = restore_dir(repo, id, target)
	if err != nil {
		return err
	}

	fmt.Printf("%q restored to %q\n", id, target)

	return nil
}
