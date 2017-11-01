package main_test

import (
  "os"
  "testing"
  "log"
  "net/http"
  "net/http/httptest"
  "encoding/json"
  "bytes"
  "strconv"
  "."
)

const tableCreationQuery = `CREATE TABLE IF NOT EXISTS products
(
  id SERIAL,
  name TEXT NOT NULL,
  price NUMERIC(10,2) NOT NULL DEFAULT 0.00,
  CONSTRAINT products_pkey PRIMARY KEY (id)
)`

var a main.App

// Ensures that database is correctly set up and cleared before running tests
func TestMain(m *testing.M) {
  main.SetEnvironmentVariables()

  a = main.App{}
  a.Initialize(
    os.Getenv("TEST_DB_USERNAME"),
    os.Getenv("TEST_DB_PASSWORD"),
    os.Getenv("TEST_DB_NAME"))

  ensureTableExists()

  code := m.Run()

  clearTable()

  os.Exit(code)
}

func ensureTableExists() {
  if _, err := a.DB.Exec(tableCreationQuery); err != nil {
    log.Fatal(err)
  }
}

func clearTable() {
  // Remember that 'a' has a DB property for the database, as per struct
  a.DB.Exec("DELETE FROM products")
  a.DB.Exec("ALTER SEQUENCE products_id_seq RESTART WITH 1")
}

func TestEmptyTable(t *testing.T) {
  clearTable()

  req, _ := http.NewRequest("GET", "/product", nil)
  response := executeRequest(req)

  checkResponseCode(t, http.StatusOK, response.Code)

  if body := response.Body.String(); body != "[]" {
    t.Errorf("Expected an empty array. Got %s", body)
  }
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
  rr := httptest.NewRecorder()
  a.Router.ServeHTTP(rr, req)
  return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
  if actual != expected {
    t.Errorf("Expected response code %d. Got %d\n", expected, actual)
  }
}

func TestGetNonExistentProduct(t *testing.T) {
  clearTable()

  req, _ := http.NewRequest("GET", "/product/11", nil)
  response := executeRequest(req)

  checkResponseCode(t, http.StatusNotFound, response.Code)

  var m map[string]string
  // Parse JSON data into format of m; stores key value pairs into the map
  json.Unmarshal(response.Body.Bytes(), &m)
  if m["error"] != "Product not found" {
    t.Errorf("Expected the 'error' key of the response to be set to 'Product not found'. Got %s.", m["error"])
  }
}

func TestCreateProduct(t *testing.T) {
  clearTable()

  payload := []byte(`{ "name": "test product", "price": 11.22 }`)

  req, _ := http.NewRequest("POST", "/product", bytes.NewBuffer(payload))
  response := executeRequest(req)

  checkResponseCode(t, http.StatusCreated, response.Code)

  var m map[string]interface{}
  json.Unmarshal(response.Body.Bytes(), &m)

  if m["name"] != "test product" {
    t.Errorf("Expected product name to be 'test product'. Got %v", m["name"])
  }

  if m["price"] != 11.22 {
    t.Errorf("Expected product price to be 11.22. Got %v", m["price"])
  }

  // the id is compared to 1.0 because JSON unmarshaling converts numbers to
  // floats, when the target is a map[string]interface{}
  if m["id"] != 1.0 {
    t.Errorf("Expected product ID to be '1'. Got %v", m["id"])
  }
}

func TestGetProduct(t *testing.T) {
  clearTable()
  addProducts(1)

  req, _ := http.NewRequest("GET", "/product/1", nil)
  response := executeRequest(req)

  checkResponseCode(t, http.StatusOK, response.Code)
}

func addProducts(count int) {
  if count < 1 {
    count = 1
  }

  for i := 0; i < count; i++ {
    a.DB.Exec("INSERT INTO products(name, price) VALUES($1, $2)", "Product " +strconv.Itoa(i), (i+1.0)*10)
  }
}

func TestUpdateProduct(t *testing.T) {
  clearTable()
  addProducts(1)

  req, _ := http.NewRequest("GET", "/product/1", nil)
  response := executeRequest(req)

  var originalProduct map[string]interface{}
  json.Unmarshal(response.Body.Bytes(), &originalProduct)

  payload := []byte(`{ "name": "updated product", "price": 22.33 }`)

  // NewBuffer prepares a Buffer to read existing data
  req, _ = http.NewRequest("PUT", "/product/1", bytes.NewBuffer(payload))
  response = executeRequest(req)

  var updatedProduct map[string]interface{}
  json.Unmarshal(response.Body.Bytes(), &updatedProduct)

  if updatedProduct["name"] != "updated product" {
    t.Errorf("Expected product name to be 'updated product'. Got %v", updatedProduct["name"])
  }

  if updatedProduct["price"] != 22.33 {
    t.Errorf("Expected product price to be '22.33'. Got %v", updatedProduct["price"])
  }

  if updatedProduct["id"] != originalProduct["id"] {
    t.Errorf("Expected the ID to remain the same (%v). Got %v", originalProduct["id"], updatedProduct["id"])
  }
}

func TestDeleteProduct(t *testing.T) {
  clearTable()
  addProducts(1)

  req, _ := http.NewRequest("GET", "/product/1", nil)
  response := executeRequest(req)
  checkResponseCode(t, http.StatusOK, response.Code)

  req, _ = http.NewRequest("DELETE", "/product/1", nil)
  response = executeRequest(req)
  checkResponseCode(t, http.StatusOK, response.Code)

  req, _ = http.NewRequest("GET", "/product/1", nil)
  response = executeRequest(req)
  checkResponseCode(t, http.StatusNotFound, response.Code)

}
