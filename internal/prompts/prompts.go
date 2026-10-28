// Package prompts keeps the user's reusable prompts.
//
// They live beside magpie's settings rather than in the Library: changing a
// prompt is not something an agent has to be synced for.
package prompts

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/filelock"
	"github.com/yetone/magpie/internal/settings"
)

// ErrNotFound is returned when a prompt id is no longer in the file.
var ErrNotFound = errors.New("prompt not found")

// Prompt is one reusable piece of text.
type Prompt struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
}

type file struct {
	Prompts []Prompt `json:"prompts"`
}

var mu filelock.Mutex

func init() {
	mu.Path = func() string { return Path() + ".lock" }
}

// Path is where prompts are kept.
func Path() string { return filepath.Join(settings.Dir(), "prompts.json") }

// List returns every prompt in the order the user made them.
func List() ([]Prompt, error) {
	mu.Lock()
	defer mu.Unlock()
	f, err := load()
	if err != nil {
		return nil, err
	}
	return slices.Clone(f.Prompts), nil
}

// Find returns one prompt.
func Find(id string) (Prompt, error) {
	mu.Lock()
	defer mu.Unlock()
	f, err := load()
	if err != nil {
		return Prompt{}, err
	}
	for _, p := range f.Prompts {
		if p.ID == id {
			return p, nil
		}
	}
	return Prompt{}, ErrNotFound
}

// Save adds a prompt, or changes the one with id. An empty id makes a new
// prompt.
func Save(id, name, text string) (Prompt, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Prompt{}, errors.New("a prompt needs a name")
	}
	if strings.TrimSpace(text) == "" {
		return Prompt{}, errors.New("a prompt needs some text")
	}
	mu.Lock()
	defer mu.Unlock()
	f, err := load()
	if err != nil {
		return Prompt{}, err
	}
	if id == "" {
		id, err = newID()
		if err != nil {
			return Prompt{}, err
		}
		p := Prompt{ID: id, Name: name, Text: text}
		f.Prompts = append(f.Prompts, p)
		if err := save(f); err != nil {
			return Prompt{}, err
		}
		return p, nil
	}
	for i := range f.Prompts {
		if f.Prompts[i].ID != id {
			continue
		}
		f.Prompts[i].Name, f.Prompts[i].Text = name, text
		if err := save(f); err != nil {
			return Prompt{}, err
		}
		return f.Prompts[i], nil
	}
	return Prompt{}, ErrNotFound
}

// Remove takes a prompt out.
func Remove(id string) error {
	mu.Lock()
	defer mu.Unlock()
	f, err := load()
	if err != nil {
		return err
	}
	for i, p := range f.Prompts {
		if p.ID != id {
			continue
		}
		f.Prompts = slices.Delete(f.Prompts, i, i+1)
		return save(f)
	}
	return ErrNotFound
}

func load() (file, error) {
	var f file
	b, err := edit.Read(Path())
	if err != nil {
		return f, err
	}
	if len(b) == 0 {
		return f, nil
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return f, fmt.Errorf("%s: %w", Path(), err)
	}
	return f, nil
}

func save(f file) error {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(Path()), 0o755); err != nil {
		return err
	}
	created := false
	if _, err := os.Stat(Path()); os.IsNotExist(err) {
		out, err := os.OpenFile(Path(), os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		created = true
	}
	if err := edit.WriteAtomic(Path(), append(b, '\n')); err != nil {
		if created {
			_ = os.Remove(Path())
		}
		return err
	}
	return nil
}

func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
