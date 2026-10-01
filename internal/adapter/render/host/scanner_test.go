package host_test

import (
	"context"
	"io/fs"
	"os/exec"
	"strings"
	"testing"

	"github.com/crispuscrew/resumegen"
	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
	"github.com/crispuscrew/resumegen/internal/adapter/render/host"
	"github.com/crispuscrew/resumegen/internal/domain"
)

func TestRender_ScannerReadingOrder(test *testing.T) {
	for _, tool := range []string{"typst", "pdftotext"} {
		if _, err := exec.LookPath(tool); err != nil {
			test.Skipf("%s unavailable; skipping PDF extraction test", tool)
		}
	}
	for _, language := range []string{"en", "ru", "empty", "wrapped"} {
		test.Run(language, func(test *testing.T) {
			directory := test.TempDir()
			skeleton, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
			if err != nil {
				test.Fatal(err)
			}
			if err := appdir.CopySkeleton(skeleton, directory, func(string, bool) bool { return true }); err != nil {
				test.Fatal(err)
			}
			data := scannerResume()
			profile := domain.Profile{Lang: language, Output: "scanner.pdf"}
			if language == "empty" {
				data = domain.ResumeData{Header: domain.Header{Name: domain.I18n{"en": "Ada Example"}}}
				profile.Lang = "en"
			}
			if language == "wrapped" {
				profile.Lang = "en"
				data.Jobs[0].Company["en"] = strings.Repeat("International ", 6) + "Example Corp"
			}
			path, pages, err := (host.Renderer{Appdir: directory}).Render(context.Background(), data, profile, (domain.Config{}).WithDefaults())
			if err != nil {
				test.Fatal(err)
			}
			if pages <= 0 || pages > 1 {
				test.Fatalf("expected a one-page fixture, got %.3f pages", pages)
			}
			for _, mode := range []string{"", "-raw", "-layout"} {
				arguments := []string{path, "-"}
				if mode != "" {
					arguments = append([]string{mode}, arguments...)
				}
				raw, err := exec.Command("pdftotext", arguments...).Output()
				if err != nil {
					test.Fatal(err)
				}
				text := strings.Join(strings.Fields(string(raw)), " ")
				if language == "empty" {
					if text != "Ada Example" {
						test.Errorf("%s: empty sections leaked into PDF: %q", mode, text)
					}
					continue
				}
				expected := []string{"Ada Example", "ada@example.com", "linkedin.com/in/ada-example", "SUMMARY"}
				headings := []string{"WORK EXPERIENCE", "PROJECTS", "TECHNICAL SKILLS", "EDUCATION"}
				if language == "ru" {
					expected[0], expected[3] = "Ада Пример", "О СЕБЕ"
					headings = []string{"ОПЫТ РАБОТЫ", "ПРОЕКТЫ", "ТЕХНИЧЕСКИЕ НАВЫКИ", "ОБРАЗОВАНИЕ"}
				}
				employer := data.Jobs[0].Company.Lang(profile.Lang)
				expected = append(expected, "efficient office workflows", headings[0], "Senior Software Engineer",
					employer+" | Jan. 2024 - Present | Berlin, Germany", "Built C++ and Go services", "Earlier Engineer",
					"Previous Corp | Jan. 2021 - Dec. 2023", "Reduced p99 latency to 40 ms", headings[1], "Portfolio Service",
					"Go, PostgreSQL | github.com/ada/portfolio", "Published a REST API", headings[2], "Languages: Go, C++", headings[3], "Example University",
					"B.Sc. Computer Science | 2020 - 2024 | Munich, Germany")
				remaining := text
				for _, field := range expected {
					position := strings.Index(remaining, field)
					if position < 0 {
						test.Fatalf("%s: missing or reordered %q in extracted PDF:\n%s", mode, field, text)
					}
					remaining = remaining[position+len(field):]
				}
			}
		})
	}
}

func scannerResume() domain.ResumeData {
	return domain.ResumeData{
		Header: domain.Header{
			Name:    domain.I18n{"en": "Ada Example", "ru": "Ада Пример"},
			Summary: domain.I18n{"en": "Builds efficient office workflows"},
			Contacts: []domain.Contact{
				{Value: "ada@example.com", Href: "mailto:ada@example.com"},
				{Value: "linkedin.com/in/ada-example", Href: "https://linkedin.com/in/ada-example"},
			},
		},
		Jobs: []domain.Job{
			{Title: domain.I18n{"en": "Senior Software Engineer"}, Company: domain.I18n{"en": "Example Corp"},
				Date: domain.I18n{"en": "Jan. 2024 - Present"}, Location: domain.I18n{"en": "Berlin, Germany"},
				Bullets: []domain.Bullet{{Text: domain.I18n{"en": "Built *C++* and *Go* services"}}}},
			{Title: domain.I18n{"en": "Earlier Engineer"}, Company: domain.I18n{"en": "Previous Corp"},
				Date:    domain.I18n{"en": "Jan. 2021 - Dec. 2023"},
				Bullets: []domain.Bullet{{Text: domain.I18n{"en": "Reduced p99 latency to *40 ms*"}}}},
		},
		Projects: []domain.Project{{Title: domain.I18n{"en": "Portfolio Service"}, Subtitle: domain.I18n{"en": "Go, PostgreSQL"},
			Detail: domain.I18n{"en": "github.com/ada/portfolio"}, Bullets: []domain.Bullet{{Text: domain.I18n{"en": "Published a REST API"}}}}},
		SkillCats: []domain.SkillCat{{Name: domain.I18n{"en": "Languages"}, Items: []domain.SkillItem{
			{Name: domain.I18n{"en": "Go"}}, {Name: domain.I18n{"en": "C++"}},
		}}},
		Edu: []domain.Edu{{Title: domain.I18n{"en": "Example University"}, Degree: domain.I18n{"en": "B.Sc. Computer Science"},
			Date: domain.I18n{"en": "2020 - 2024"}, Location: domain.I18n{"en": "Munich, Germany"}}},
	}
}
