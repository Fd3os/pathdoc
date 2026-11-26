//go:build repro

package repro

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Web toolchains, checked and fine: npm, Vite, Next.js, Tailwind, Prisma and
// sharp build in all three folders (Node uses Unicode paths). This test keeps
// it true: the week a new version breaks in café or ابوي, it fails and the
// break becomes a rule.
func TestRepro_web_tools_fine(t *testing.T) {
	need := func(name string) string {
		p, err := exec.LookPath(name)
		if err != nil {
			if os.Getenv("PATHDOC_REPRO_STRICT") == "1" {
				t.Fatalf("%s is not installed", name)
			}
			t.Skipf("%s is not installed", name)
		}
		return p
	}
	npm, npx, node := need("npm"), need("npx"), need("node")
	base := lab(t)
	for _, f := range folders {
		dir := filepath.Join(base, f.name, "web")
		write(t, filepath.Join(dir, "package.json"), `{"name":"web","version":"1.0.0","private":true,"dependencies":{"vite":"^7","next":"^16","react":"^19","react-dom":"^19","tailwindcss":"^4","@tailwindcss/cli":"^4","prisma":"^6","@prisma/client":"^6","sharp":"^0.34"}}`)
		write(t, filepath.Join(dir, "index.html"), `<!doctype html><html><body><div id="app"></div><script type="module" src="/main.js"></script></body></html>`)
		write(t, filepath.Join(dir, "main.js"), "document.querySelector('#app').textContent = 'hello-ok'\n")
		write(t, filepath.Join(dir, "app", "layout.js"), "export default function L({children}){return <html><body>{children}</body></html>}\n")
		write(t, filepath.Join(dir, "app", "page.js"), "export default function P(){return <p>hello-ok</p>}\n")
		write(t, filepath.Join(dir, "input.css"), "@import \"tailwindcss\";\n")
		write(t, filepath.Join(dir, "prisma", "schema.prisma"), "datasource db {\n  provider = \"sqlite\"\n  url = \"file:./dev.db\"\n}\ngenerator client {\n  provider = \"prisma-client-js\"\n}\nmodel User {\n  id Int @id\n}\n")
		write(t, filepath.Join(dir, "sharp-test.js"), "require('sharp')({create:{width:8,height:8,channels:3,background:'#09f'}}).png().toFile('out.png').then(()=>console.log('hello-ok'),e=>{console.error(e);process.exit(1)})\n")
		env := []string{"NEXT_TELEMETRY_DISABLED=1", "PRISMA_HIDE_UPDATE_MESSAGE=1"}
		pass(t, f.name+": npm install", run(t, dir, env, npm, "install", "--no-audit", "--no-fund"), "added")
		pass(t, f.name+": vite build", run(t, dir, env, npx, "vite", "build"), "built in")
		pass(t, f.name+": next build", run(t, dir, env, npx, "next", "build"), "Compiled")
		pass(t, f.name+": tailwind", run(t, dir, env, npx, "@tailwindcss/cli", "-i", "input.css", "-o", "out.css"), "Done")
		pass(t, f.name+": prisma generate", run(t, dir, env, npx, "prisma", "generate"), "Generated Prisma Client")
		pass(t, f.name+": sharp", run(t, dir, env, node, "sharp-test.js"), "hello-ok")
	}
}

// node-gyp builds native add-ons (bcrypt, sqlite3, canvas…) with MSVC and
// Python — the web toolchain most likely to break on a path. Needs Visual
// Studio's C++ tools, which GitHub's Windows runners have.
func TestRepro_node_gyp(t *testing.T) {
	npx, err := exec.LookPath("npx")
	if err != nil {
		t.Skip("npx is not installed")
	}
	if _, err := os.Stat(`C:\Program Files (x86)\Microsoft Visual Studio\Installer\vswhere.exe`); err != nil {
		if os.Getenv("PATHDOC_REPRO_STRICT") == "1" {
			t.Fatal("Visual Studio is not installed")
		}
		t.Skip("Visual Studio's C++ tools are not installed")
	}
	base := lab(t)
	for _, f := range folders {
		dir := filepath.Join(base, f.name, "addon")
		write(t, filepath.Join(dir, "binding.gyp"), `{"targets":[{"target_name":"hello","sources":["hello.c"]}]}`)
		write(t, filepath.Join(dir, "hello.c"), "#include <node_api.h>\nstatic napi_value Init(napi_env env, napi_value exports) { return exports; }\nNAPI_MODULE(NODE_GYP_MODULE_NAME, Init)\n")
		r := run(t, dir, nil, npx, "--yes", "node-gyp@11", "rebuild")
		t.Logf("%s: node-gyp exit %d\n%s", f.name, r.code, tail(r.out))
		pass(t, f.name+": node-gyp rebuild", r, "ok")
	}
}
