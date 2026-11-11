package codie

import (
	"os"
	"strings"
	"testing"
)

// each kind of program is recognised from typical code
func TestPurpose(t *testing.T) {
	cases := []struct{ want, code string }{
		{"micro", "// Blink the LED\nvoid setup() {\n  pinMode(13, OUTPUT);\n}\nvoid loop() {\n  digitalWrite(13, HIGH);\n  delay(500);\n  digitalWrite(13, LOW);\n  delay(500);\n}\n"},
		{"webserver", "from flask import Flask, jsonify\napp = Flask(__name__)\n\n@app.route('/hello')\ndef hello():\n    return jsonify(msg='hi')\n\nif __name__ == '__main__':\n    app.run()\n"},
		{"data", "import pandas as pd\nimport matplotlib.pyplot as plt\ndf = pd.read_csv('sales.csv')\nprint(df.groupby('month').sum())\ndf.plot()\nplt.show()\n"},
		{"interactive", "name = input('What is your name? ')\nage = int(input('How old are you? '))\nprint('Hello', name)\nif age > 12:\n    print('You are a teenager or older')\n"},
		{"webpage", "const btn = document.getElementById('go');\nbtn.addEventListener('click', () => {\n  document.querySelector('#out').textContent = 'Clicked!';\n});\n"},
		{"fetch", "import requests\nr = requests.get('https://api.example.com/weather')\ndata = r.json()\nprint(data['temp'])\n"},
		{"cli", "package main\n\nimport (\n\t\"flag\"\n\t\"fmt\"\n)\n\nfunc main() {\n\tn := flag.Int(\"n\", 3, \"how many\")\n\tflag.Parse()\n\tfmt.Println(*n)\n}\n"},
		{"keyboard", "const uint16_t PROGMEM keymaps[][MATRIX_ROWS][MATRIX_COLS] = {\n  [0] = LAYOUT(KC_A, KC_B, KC_C)\n};\n"},
	}
	for _, c := range cases {
		r := Analyse(c.code, "")
		if len(r.Purpose.Kinds) == 0 || r.Purpose.Kinds[0].ID != c.want {
			got := "nothing"
			if len(r.Purpose.Kinds) > 0 {
				got = r.Purpose.Kinds[0].ID
			}
			t.Errorf("want %s, got %s: %s", c.want, got, r.Purpose.Headline)
		}
		if os.Getenv("CODIE_SHOW_PURPOSE") != "" {
			t.Logf("%s: [%s] %s | flow %v", c.want, r.Purpose.Sure, r.Purpose.Headline, r.Purpose.Flow)
		}
	}
	// every built-in example gets a headline
	for _, s := range allExamples() {
		r := Analyse(s.Code, "")
		if r.Purpose.Headline == "" || strings.Contains(r.Purpose.Headline, "  ") {
			t.Errorf("%s: headline %q", s.Name, r.Purpose.Headline)
		}
		if os.Getenv("CODIE_SHOW_PURPOSE") != "" {
			t.Logf("%-40s [%s] %s", s.Name, r.Purpose.Sure, r.Purpose.Headline)
		}
	}
}
