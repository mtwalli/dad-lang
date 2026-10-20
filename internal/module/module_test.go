package module

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func reader(files map[string]string) Reader {
	return func(p string) (string, error) {
		if s, ok := files[filepath.ToSlash(p)]; ok {
			return s, nil
		}
		return "", errors.New("not found")
	}
}

func TestLoad(t *testing.T) {
	p := Load("app/main.dad", reader(map[string]string{
		"app/main.dad":       "استورد أ من \"./أدوات\"\nاستورد ب من \"نماذج/نقطة\"\nاطبع(أ)\n",
		"app/أدوات.dad":      "استورد ب من \"./نماذج/نقطة.dad\"\nثابت أ = 1\n",
		"app/نماذج/نقطة.dad": "ثابت ب = 2\n",
	}))
	var names []string
	for _, m := range p.Modules {
		names = append(names, filepath.ToSlash(m.Path))
	}
	if got := strings.Join(names, " "); got != "app/نماذج/نقطة.dad app/أدوات.dad app/main.dad" {
		t.Errorf("order: %s", got)
	}
	if len(p.Diags()) != 0 {
		t.Errorf("diags: %v", p.Diags())
	}
	if len(p.Main.Deps) != 2 || len(p.Modules[1].Deps) != 1 {
		t.Errorf("deps: %v %v", p.Main.Deps, p.Modules[1].Deps)
	}
}

func TestProblems(t *testing.T) {
	p := Load("أ.dad", reader(map[string]string{
		"أ.dad": "استورد س من \"ب\"\nاستورد ص من \"مفقود\"\n",
		"ب.dad": "استورد ع من \"ج\"\nثابت س = 1\n",
		"ج.dad": "استورد أ من \"أ\"\nثابت ع = 1\n",
	}))
	ds := p.Diags()
	if len(ds) != 2 {
		t.Fatalf("diags: %v", ds)
	}
	if ds[0].Path != "ج.dad" || !strings.Contains(ds[0].Msg, "«أ» يستورد «ب»، و«ب» يستورد «ج»، و«ج» يستورد «أ»") {
		t.Errorf("circle: %v", ds[0])
	}
	if ds[1].Path != "أ.dad" || !strings.Contains(ds[1].Msg, "لا يوجد ملف «مفقود»") || ds[1].Pos.Line != 2 {
		t.Errorf("missing: %v", ds[1])
	}
}
