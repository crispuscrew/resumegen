package render

import (
	"github.com/crispuscrew/resumegen/internal/model"

	"log"
	"encoding/json"
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"os"
	"os/exec"
	"errors"
	"strconv"
	"strings"
)

func Render(cfg model.Config, data model.ResumeData, profile model.Profile, appDirPath string) (string, float64) {
	datagen, err := build(data, profile)
	if err != nil { log.Fatalf("Datagen typ error: %v", err); return "", 0.0}

	outPath, pages, err := compile(datagen, appDirPath, cfg, profile)
	if err != nil { log.Fatalf("Cannot resolve path to rendered path: %v", err); return "", 0.0}
	return outPath, pages
}

const (
	dirPerm  fs.FileMode = 0o755 // rwxr-xr-x                                          
    filePerm fs.FileMode = 0o644 // rw-r--r--
)

func compile(dataGen []byte, appDirPath string, cfg model.Config, profile model.Profile) (string, float64, error) {
	dataGenPath := filepath.Join(appDirPath, "templates", "data_gen.typ")

	err := os.WriteFile(dataGenPath, dataGen, filePerm)
	if err != nil {return "", 0.0, err}
	defer func() { _ = os.Remove(dataGenPath) }()

	outPath := filepath.Join(appDirPath, cfg.Paths.OutputDir, profile.Output)
	typPath := filepath.Join(appDirPath, "templates", "resume.typ")

	err = os.MkdirAll(filepath.Dir(outPath), dirPerm)
	if err != nil { return "", 0.0, err }

	cmd := exec.Command(cfg.Paths.TypstBin, "compile", typPath, outPath)
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {return "", 0.0, err}

	pages, err := pageFloatCount(typPath, cfg)

	return outPath, pages, err
}

func pageFloatCount(pathToTyp string, cfg model.Config) (float64, error) {
	out, err := exec.Command(cfg.Paths.TypstBin, "query",
			pathToTyp, "<end-marker>", "--field", "value",
		).Output()
	if err != nil {return 0.0, err}

	type typstPos struct {
		Page int     	`json:"page"`
		X    string 	`json:"x"`
		Y    string 	`json:"y"`
	}
	var positions []typstPos
	if err = json.Unmarshal(out, &positions); err != nil { return 0, err }
	if len(positions) == 0 { return 0, errors.New("end-marker not found") }

	pos := positions[0]
	y, err := strconv.ParseFloat(strings.TrimSuffix(pos.Y, "pt"), 64)
	if err != nil { return 0.0, err }

	const pageHeight = 792
	// pageHeight in pt for us-letter = 792
	return float64(pos.Page - 1) + y / pageHeight, nil 
}

func build(data model.ResumeData, profile model.Profile) ([]byte, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "#let r-lang = %q\n", profile.Lang)
	fmt.Fprintf(&buf, "#let r-name = %q\n", data.Header.Name)
	fmt.Fprintf(&buf, "#let r-summary = [%s]\n", data.Header.Summary.Lang(profile.Lang))

	//contacts
	buf.WriteString("#let r-contacts = (\n")
	for _, contact := range data.Header.Contacts {
		if contact.Lang == "" || contact.Lang == profile.Lang {
			fmt.Fprintf(&buf, "	(value : %q, href : %q), \n", contact.Value, contact.Href)
		}
	}
	buf.WriteString(")\n")

	//jobs
	buf.WriteString("#let r-jobs = (\n")
	for _, job := range data.Jobs {
		if job.Reason != model.Included {continue}

		fmt.Fprintf(&buf, "	(title : %q, date : %q, company : %q, location : %q, bullets : (",
			job.Title.Lang(profile.Lang), job.Date.Lang(profile.Lang),
			job.Company, job.Location.Lang(profile.Lang))
		for _, bullet := range job.Bullets {
			if bullet.Reason != model.Included {continue}
			text := bullet.En
			if profile.Lang == "ru" { text = bullet.Ru }
			fmt.Fprintf(&buf, "\n		[%s],", text)
		}
		buf.WriteString(")),\n")
	}
	buf.WriteString(")\n")

	//projects
	buf.WriteString("#let r-projects = (\n")
	for _, project := range data.Projects {
		if project.Reason != model.Included {continue}

		fmt.Fprintf(&buf, "	(title : %q, date : %q, subtitle : %q, detail : %q, bullets : (",
			project.Title, project.Date, project.Subtitle, project.Detail)
		for _, bullet := range project.Bullets {
			if bullet.Reason != model.Included {continue}
			text := bullet.En
			if profile.Lang == "ru" { text = bullet.Ru }
			fmt.Fprintf(&buf, "\n		[%s],", text)
		}
		buf.WriteString(")),\n")
	}
	buf.WriteString(")\n")

	//skills
	buf.WriteString("#let r-skills = (\n")
	for _, skillsCat := range data.SkillCats {
		if skillsCat.Reason != model.Included {continue}

		fmt.Fprintf(&buf, "	(category : %q, items : (", skillsCat.Name.Lang(profile.Lang))
		for _, skill := range skillsCat.Items {
			if skill.Reason != model.Included {continue}
			fmt.Fprintf(&buf, "[%s],", skill.Name)
		}
		buf.WriteString(")),\n")
	}
	buf.WriteString(")\n")

	//edu
	buf.WriteString("#let r-edu = (\n")
	for _, edu := range data.Edu {
		fmt.Fprintf(&buf, "	(title : %q, degree : %q, location : %q, date : %q),\n",
			edu.Title.Lang(profile.Lang), edu.Degree.Lang(profile.Lang),
			edu.Location.Lang(profile.Lang), edu.Date.Lang(profile.Lang))
	}
	buf.WriteString(")\n")

	return buf.Bytes(), nil
}