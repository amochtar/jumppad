package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

type Book struct {
	config *resources.Book
	log    clients.Logger
}

func NewBook(b *resources.Book, l clients.Logger) *Book {
	return &Book{b, l}
}

func (b *Book) Create() error {
	b.log.Info(fmt.Sprintf("Creating %s", strings.Title(string(b.config.Metadata().Type))), "ref", b.config.Metadata().Name)

	book := resources.IndexBook{
		Title: b.config.Title,
	}

	libraryPath := utils.GetLibraryFolder("", 0775)
	bookPath := filepath.Join(libraryPath, "content", b.config.Name)

	for _, bc := range b.config.Chapters {
		cr, err := b.config.ParentConfig.FindResource(bc)
		if err != nil {
			return fmt.Errorf("Unable to create book %s, could not find chapter %s", b.config.Metadata().Name, bc)
		}

		c := cr.(*resources.Chapter)

		chapterPath := filepath.Join(bookPath, c.Name)

		chapter := resources.IndexChapter{
			Title: c.Title,
		}

		for _, p := range c.Pages {
			err = b.writePage(chapterPath, p)
			if err != nil {
				return err
			}

			page := resources.IndexPage{
				Title: p.Title,
				URI:   fmt.Sprintf("/%s/%s/%s", b.config.Name, c.Name, p.Name),
			}

			chapter.Pages = append(chapter.Pages, page)
		}

		book.Chapters = append(book.Chapters, chapter)
	}

	b.config.Index = book

	return nil
}

func (b *Book) Destroy() error {
	return nil
}

func (b *Book) Lookup() ([]string, error) {
	return nil, nil
}

func (b *Book) Refresh() error {
	b.log.Debug("Refresh Book", "ref", b.config.Name)

	libraryPath := utils.GetLibraryFolder("", 0775)
	bookPath := filepath.Join(libraryPath, "content", b.config.Name)

	for _, bc := range b.config.Chapters {
		cr, err := b.config.ParentConfig.FindResource(bc)
		if err != nil {
			return err
		}

		c := cr.(*resources.Chapter)

		chapterPath := filepath.Join(bookPath, c.Name)

		for _, p := range c.Pages {
			err = b.writePage(chapterPath, p)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Book) Changed() (bool, error) {
	c.log.Debug("Checking changes", "ref", c.config.Name)

	return false, nil
}

func (b *Book) writePage(chapterPath string, page resources.Page) error {
	os.MkdirAll(chapterPath, 0755)
	os.Chmod(chapterPath, 0755)

	if len(page.Tasks) > 0 {
		r, _ := regexp.Compile("<Task id=\"(?P<id>.*)\">")
		match := r.FindStringSubmatch(page.Content)
		result := map[string]string{}
		for i, name := range r.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}

		if len(match) > 0 {
			taskID := result["id"]
			resourceID := fmt.Sprintf("<Task id=\"%s\">", page.Tasks[taskID])
			page.Content = r.ReplaceAllString(page.Content, resourceID)
		}
	}

	pageFile := fmt.Sprintf("%s.mdx", page.Name)
	pagePath := filepath.Join(chapterPath, pageFile)
	err := os.WriteFile(pagePath, []byte(page.Content), 0755)
	if err != nil {
		return fmt.Errorf("Unable to write page %s to disk at %s", page.Name, pagePath)
	}

	return nil
}

type Chapter struct {
	config *resources.Chapter
	log    clients.Logger
}

func NewChapter(c *resources.Chapter, l clients.Logger) *Chapter {
	return &Chapter{c, l}
}

func (c *Chapter) Create() error {
	c.log.Info(fmt.Sprintf("Creating %s", strings.Title(string(c.config.Metadata().Type))), "ref", c.config.Metadata().Name)

	tasks := []string{}

	for _, p := range c.config.Pages {
		for _, task := range p.Tasks {
			tasks = append(tasks, task)
		}
	}

	c.config.Tasks = tasks

	return nil
}

func (c *Chapter) Destroy() error {
	return nil
}

func (c *Chapter) Lookup() ([]string, error) {
	return nil, nil
}

func (c *Chapter) Refresh() error {
	return nil
}

func (c *Chapter) Changed() (bool, error) {
	c.log.Debug("Checking changes", "ref", c.config.Name)

	return false, nil
}
