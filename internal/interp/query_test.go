package interp

import (
	"io"
	"testing"

	"github.com/mtwalli/dad-lang/internal/parser"
)

func TestCompileWhere(t *testing.T) {
	src := `جدول مهمة {
  عنوان: نص
  منجزة = خطأ
  أولوية = 1
  ملاحظة: نص؟
}
ثابت حد = 2
`
	cases := map[string]string{
		"{ هو.منجزة }": "منجزة",
		"{ ليس هو.منجزة و هو.أولوية >= حد }":        "(not منجزة and أولوية ge 2)",
		"{ م => م.عنوان == \"أ\" أو 3 > م.أولوية }": `(عنوان eq "أ" or أولوية lt 3)`,
		"{ هو.ملاحظة != عدم }":                      "ملاحظة ne <nil>",
		"{ هو.أولوية == -1 }":                       "أولوية eq -1",
		// These run on every row instead.
		"{ هو.عنوان.يحتوي(\"أ\") }": "",
		"{ هو.أولوية * 2 > 3 }":     "",
		"{ هو.أولوية > حد() }":      "",
		"{ هو.عنوان == 5 }":         "",
		"{ هو.ملاحظة > \"أ\" }":     "",
		"{ أ، ب => أ.منجزة }":       "",
	}
	for lam, want := range cases {
		file, diags := parser.ParseFile(src + "ثابت ش = " + lam + "\n")
		if file == nil {
			t.Fatalf("%s: %v", lam, diags)
		}
		in := New(io.Discard)
		if err := in.Run(file); err != nil {
			t.Fatal(err)
		}
		c, ok := in.compileWhere(in.globals.lookup("ش").val, in.tables[0])
		got := ""
		if ok {
			got = c.String()
		}
		if got != want {
			t.Errorf("%s: got %q, want %q", lam, got, want)
		}
	}
}
