package util

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFiles(t *testing.T) {
	rootPath := t.TempDir()
	t.Log("TestFiles: " + rootPath)

	assert.Nil(t, os.Mkdir(path.Join(rootPath, "subDir1"), 0755))
	assert.Nil(t, os.Mkdir(path.Join(rootPath, "subDir2"), 0755))
	assert.Nil(t, os.Mkdir(path.Join(rootPath, "subDir3"), 0755))
	assert.Nil(t, os.WriteFile(path.Join(rootPath, "subDir1", "test1a"), []byte("Hello1a"), 0644))
	assert.Nil(t, os.WriteFile(path.Join(rootPath, "subDir1", "test1b"), []byte("Hello1b"), 0644))
	assert.Nil(t, os.WriteFile(path.Join(rootPath, "subDir2", "test2"), []byte("Hello2"), 0644))
	assert.Nil(t, os.WriteFile(path.Join(rootPath, "subDir3", "test3"), []byte("Hello3"), 0644))

	t.Run("list files", func(tt *testing.T) {
		files, err := ListFiles(path.Join(rootPath, "subDir[13]"))
		assert.Nil(t, err)
		assert.Equal(t, []string{
			path.Join(rootPath, "subDir1", "test1a"),
			path.Join(rootPath, "subDir1", "test1b"),
			path.Join(rootPath, "subDir3", "test3"),
		}, files)
	})

	dir3, err3 := os.Open(path.Join(rootPath, "subDir3"))
	assert.Nil(t, err3)

	t.Run("read file at", func(tt *testing.T) {
		content, err := ReadFileAt(dir3, "test3")
		assert.Nil(t, err)
		assert.Equal(t, string(content), "Hello3")
	})

	t.Run("unlink file at", func(t *testing.T) {
		assert.Nil(t, UnlinkFileAt(dir3, "test3"))
		_, err := ReadFileAt(dir3, "test3")
		assert.NotNil(t, err)
	})

	t.Run("write file at", func(t *testing.T) {
		assert.Nil(t, WriteFileAt(dir3, "test4", []byte("Hello4"), 0644))
		content4, _ := ReadFileAt(dir3, "test4")
		assert.Equal(t, "Hello4", string(content4))
	})
}
