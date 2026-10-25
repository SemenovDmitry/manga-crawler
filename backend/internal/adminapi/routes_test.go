package adminapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// TestRoutesMatchOpenAPI — контрактный тест: каждый path+method из
// docs/openapi.yaml обязан быть зарегистрирован в роутере newEngine.
// Не даёт добавить эндпоинт в схему и забыть про роут (и наоборот).
func TestRoutesMatchOpenAPI(t *testing.T) {
	const schemaPath = "../../docs/openapi.yaml"

	schema, err := openapi3.NewLoader().LoadFromFile(schemaPath)
	if err != nil {
		t.Fatalf("загрузка %s: %v", schemaPath, err)
	}

	router := newEngine(&fakeStore{}, testConfig())

	// gin маршруты: GET|PATCH /admin-api/manga/:id -> {GET, PATCH}
	registered := map[string]map[string]bool{}
	for _, ri := range router.Routes() {
		methods, ok := registered[ri.Path]
		if !ok {
			methods = map[string]bool{}
			registered[ri.Path] = methods
		}
		methods[ri.Method] = true
	}

	// gin использует :param, OpenAPI — {param}; приводим к виду gin.
	toGinPath := func(p string) string {
		open, close := strings.IndexByte(p, '{'), strings.IndexByte(p, '}')
		for open != -1 && close != -1 {
			p = p[:open] + ":" + p[open+1:close] + p[close+1:]
			open, close = strings.IndexByte(p, '{'), strings.IndexByte(p, '}')
		}
		return p
	}

	for _, path := range schema.Paths.InMatchingOrder() {
		item := schema.Paths.Find(path)
		ginPath := "/admin-api" + toGinPath(path)
		methods, ok := registered[ginPath]
		if !ok {
			t.Errorf("роут %s есть в схеме, но не зарегистрирован в роутере", ginPath)
			continue
		}
		for method := range item.Operations() {
			if !methods[strings.ToUpper(method)] {
				t.Errorf("метод %s %s есть в схеме, но не зарегистрирован в роутере", strings.ToUpper(method), ginPath)
			}
		}
	}

	// Обратное направление: в роутере нет лишних маршрутов.
	if got, want := len(registered), len(schema.Paths.InMatchingOrder()); got != want {
		t.Errorf("в роутере %d путей, в схеме %d — расхождение", got, want)
	}
}

// testConfig — валидный конфиг для тестов (хэш пароля "secret").
func testConfig() Config {
	cfg := ConfigFromEnv()
	cfg.JWTSecret = []byte("test-secret-0123456789abcdef0123456789")
	cfg.Username = "admin"
	cfg.PasswordHash = testBcryptHash
	return cfg
}

// testBcryptHash — bcrypt("test-password"), cost 10.
const testBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// schemaPathsInSync — страховка от переименования docs/openapi.yaml.
func TestOpenAPISchemaFileExists(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "docs", "openapi.yaml")); err != nil {
		t.Fatalf("схема контракта недоступна: %v", err)
	}
}

// jsonBody — хелпер для сборки JSON-тела запроса в тестах.
func jsonBody(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}
