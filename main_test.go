package main_test

import (
	"bytes"
	"encoding/json"
	"github.com/franziseidl/cdas_semaphoreExample"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
)

var a main.App

func TestMain(m *testing.M) {
	a.Initialize(
		os.Getenv("APP_DB_USERNAME"),
		os.Getenv("APP_DB_PASSWORD"),
		os.Getenv("APP_DB_NAME"),
		os.Getenv("APP_DB_PORT"))

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

const tableCreationQuery = `CREATE TABLE IF NOT EXISTS products
(
    id SERIAL,
    name TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    CONSTRAINT products_pkey PRIMARY KEY (id)
)`

func clearTable() {
	a.DB.Exec("DELETE FROM products")
	a.DB.Exec("ALTER SEQUENCE products_id_seq RESTART WITH 1")
}

func TestEmptyTable(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/products", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); body != "[]" {
		t.Errorf("Expected an empty array. Got %s", body)
	}
}
func TestGetNonExistentProduct(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/product/11", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response.Code)

	var m map[string]string
	json.Unmarshal(response.Body.Bytes(), &m)
	if m["error"] != "Product not found" {
		t.Errorf("Expected the 'error' key of the response to be set to 'Product not found'. Got '%s'", m["error"])
	}
}
func TestCreateProduct(t *testing.T) {

	clearTable()

	var jsonStr = []byte(`{"name":"test product", "price": 11.22}`)
	req, _ := http.NewRequest("POST", "/product", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	response := executeRequest(req)
	checkResponseCode(t, http.StatusCreated, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["name"] != "test product" {
		t.Errorf("Expected product name to be 'test product'. Got '%v'", m["name"])
	}

	if m["price"] != 11.22 {
		t.Errorf("Expected product price to be '11.22'. Got '%v'", m["price"])
	}

	// the id is compared to 1.0 because JSON unmarshaling converts numbers to
	// floats, when the target is a map[string]interface{}
	if m["id"] != 1.0 {
		t.Errorf("Expected product ID to be '1'. Got '%v'", m["ID"])
	}
}
func TestGetProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/product/1", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)
}
func TestUpdateProduct(t *testing.T) {

	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/product/1", nil)
	response := executeRequest(req)
	var originalProduct map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &originalProduct)

	var jsonStr = []byte(`{"name":"test product - updated name", "price": 11.22}`)
	req, _ = http.NewRequest("PUT", "/product/1", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	response = executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["id"] != originalProduct["id"] {
		t.Errorf("Expected the id to remain the same (%v). Got %v", originalProduct["id"], m["id"])
	}

	if m["name"] == originalProduct["name"] {
		t.Errorf("Expected the name to change from '%v' to '%v'. Got '%v'", originalProduct["name"], m["name"], m["name"])
	}

	if m["price"] == originalProduct["price"] {
		t.Errorf("Expected the price to change from '%v' to '%v'. Got '%v'", originalProduct["price"], m["price"], m["price"])
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
func TestGetProductsByName(t *testing.T) {
	clearTable()
	addProduct(main.Product{Name: "Testproduct", Price: 12.34})
	addProduct(main.Product{Name: "Testproduct 2", Price: 12.35})
	addProduct(main.Product{Name: "Testproduct 3", Price: 12.36})
	addProduct(main.Product{Name: "Testproduct", Price: 12.50})

	var jsonStr = []byte(`{"name": "Testproduct"}`)
	req, _ := http.NewRequest("POST", "/product/filterByName", bytes.NewBuffer(jsonStr))
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	var m []main.Product
	json.Unmarshal(response.Body.Bytes(), &m)

	if len(m) != 2 {
		t.Errorf("Expected to find 2 products. Got '%v'", len(m))
	}
	if len(m) == 2 {
		var first = m[0]
		var second = m[1]
		if first.Name != "Testproduct" {
			t.Errorf("Expected first product name to be 'Testproduct'. Got '%v'", first.Name)
		}

		if first.Price != 12.34 {
			t.Errorf("Expected first product price to be '12.34'. Got '%v'", first.Price)
		}
		if second.Name != "Testproduct" {
			t.Errorf("Expected second product name to be Testproduct'. Got '%v'", second.Name)
		}

		if second.Price != 12.50 {
			t.Errorf("Expected second product price to be '12.50'. Got '%v'", second.Price)
		}
	}

}
func TestGetProductsByPrice(t *testing.T) {
	clearTable()
	addProduct(main.Product{Name: "Testproduct 1", Price: 10})
	addProduct(main.Product{Name: "Testproduct 2", Price: 20})
	addProduct(main.Product{Name: "Testproduct 3", Price: 30})
	addProduct(main.Product{Name: "Testproduct 4", Price: 50})
	addProduct(main.Product{Name: "Testproduct 5", Price: 40})

	var jsonStr = []byte(`{"minPrice": 20, "maxPrice": 40}`)
	req, _ := http.NewRequest("POST", "/product/filterByPrice", bytes.NewBuffer(jsonStr))
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	var m []main.Product
	json.Unmarshal(response.Body.Bytes(), &m)

	if len(m) != 3 {
		t.Errorf("Expected 3 products. Got %v", len(m))
	}
	if len(m) == 3 {
		var first = m[0]
		var second = m[1]
		var third = m[2]
		if first.Name != "Testproduct 2" {
			t.Errorf("Expected first product to be 'Testproduct 2'. Got '%v'", first.Name)
		}
		if first.Price != 20 {
			t.Errorf("Expected first product price to be '20'. Got '%v'", first.Price)
		}
		if second.Name != "Testproduct 3" {
			t.Errorf("Expected second product to be 'Testproduct 3'. Got '%v'", second.Name)
		}
		if second.Price != 30 {
			t.Errorf("Expected second product price to be '30'. Got '%v'", second.Price)
		}
		if third.Name != "Testproduct 5" {
			t.Errorf("Expected third product to be 'Testproduct 5'. Got '%v'", third.Name)
		}
		if third.Price != 40 {
			t.Errorf("Expected third product price to be '40'. Got '%v'", third.Price)
		}
	}

}
func TestDuplicateProduct(t *testing.T) {
	clearTable()
	addProduct(main.Product{Name: "Testproduct 1", Price: 12.34})
	addProduct(main.Product{Name: "Testproduct 2", Price: 55})
	addProduct(main.Product{Name: "Testproduct 3", Price: 100})

	var jsonStr = []byte(`{"originId": 1, "newName": "duplicate 1" }`)
	req, _ := http.NewRequest("POST", "/product/duplicate", bytes.NewBuffer(jsonStr))
	response := executeRequest(req)
	checkResponseCode(t, http.StatusCreated, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["Id"] == 1 {
		t.Errorf("Expected product to have an id other 1.  But got '%v'", m["Id"])
	}
	if m["name"] != "duplicate 1" {
		t.Errorf("Expected product name to be 'duplicate 1'. Got '%v'", m["name"])
	}

	if m["price"] != 12.34 {
		t.Errorf("Expected product price to be '12.34'. Got '%v'", m["price"])
	}
}
func TestDuplicateProduct_NotExisting_ShouldReturn404(t *testing.T) {
	clearTable()
	addProduct(main.Product{Name: "Testproduct 1", Price: 12.34})
	addProduct(main.Product{Name: "Testproduct 2", Price: 55})
	addProduct(main.Product{Name: "Testproduct 3", Price: 100})

	var jsonStr = []byte(`{"originId": 5, "newName": "duplicate 1" }`)
	req, _ := http.NewRequest("POST", "/product/duplicate", bytes.NewBuffer(jsonStr))
	response := executeRequest(req)
	checkResponseCode(t, http.StatusNotFound, response.Code)

	var m map[string]string
	json.Unmarshal(response.Body.Bytes(), &m)
	if m["error"] != "Product not found" {
		t.Errorf("Expected the 'error' key of the response to be set to 'Product not found'. Got '%s'", m["error"])
	}
}

func addProduct(p main.Product) error {
	err := a.DB.QueryRow(
		"INSERT INTO products(name, price) VALUES($1, $2) RETURNING id",
		p.Name, p.Price).Scan(&p.ID)

	if err != nil {
		return err
	}

	return nil
}
func getProduct(id int) (main.Product, error) {
	var p main.Product
	a.DB.QueryRow("SELECT name, price FROM products WHERE id=$1",
		p.ID).Scan(&p.Name, &p.Price)
	return p, nil
}
func addProducts(count int) {
	if count < 1 {
		count = 1
	}

	for i := 0; i < count; i++ {
		a.DB.Exec("INSERT INTO products(name, price) VALUES($1, $2)", "product "+strconv.Itoa(i), (i+1.0)*10)
	}
}
func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)

	return rr
}
func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
