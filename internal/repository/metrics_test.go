package repository_test

import (
	"fmt"
	"strconv"

	"github.com/andynikk/advancedmetrics/internal/encoding"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

func ExampleGauge_Type() {
	var g repository.Gauge = 0.01
	fmt.Print(g.Type())

	// Output:
	// gauge
}

func ExampleGauge_String() {
	var g repository.Gauge = 0.01
	fmt.Print(g.String())

	// Output:
	// 0.01
}

func ExampleGauge_GetMetrics() {

	mType := "gauge"
	id := "TestGauge"
	hashKey := "TestHash"

	g := repository.Gauge(0.01)

	mt := g.GetMetrics(mType, id, hashKey)
	msg := fmt.Sprintf("MType: %s, ID: %s, Value: %f, Delta: %d, Hash: %s",
		mt.MType, mt.ID, *mt.Value, 0, mt.Hash)
	fmt.Print(msg)

	// Output:
	// MType: gauge, ID: TestGauge, Value: 0.010000, Delta: 0, Hash: 4e5d8a0e257dd12355b15f730591dddd9e45e18a6ef67460a58f20edc12c9465
}

func ExampleGauge_Set() {
	var g repository.Gauge
	var f = 0.01
	v := encoding.Metrics{
		ID:    "",
		MType: "",
		Value: &f,
		Hash:  "",
	}
	g.Set(v)
	fmt.Print(g)

	// Output:
	// 0.01
}

func ExampleGauge_SetFromText() {

	metValue := "0.01"

	predVal, _ := strconv.ParseFloat(metValue, 64)
	g := repository.Gauge(predVal)

	fmt.Print(g)

	// Output:
	// 0.01
}

////////////////////////////////////////////////////////////

func ExampleCounter_String() {
	var c repository.Counter = 58
	fmt.Println(c.String())

	// Output:
	// 58
}

func ExampleCounter_Type() {
	var c repository.Counter = 58
	fmt.Print(c.Type())

	// Output:
	// counter
}

func ExampleCounter_GetMetrics() {

	mType := "counter"
	id := "TestCounter"
	hashKey := "TestHash"

	var c repository.Counter = 58

	mt := c.GetMetrics(mType, id, hashKey)
	msg := fmt.Sprintf("MType: %s, ID: %s, Value: %v, Delta: %d, Hash: %s",
		mt.MType, mt.ID, 0, *mt.Delta, mt.Hash)
	fmt.Print(msg)

	// Output:
	// MType: counter, ID: TestCounter, Value: 0, Delta: 58, Hash: 29bd8e4bde7ec6302393fe3f7954895a65f4d4b22372d00a35fc1adbcc2ec239
}

func ExampleCounter_Set() {
	var c repository.Counter
	var i int64 = 58
	v := encoding.Metrics{
		ID:    "",
		MType: "",
		Delta: &i,
		Hash:  "",
	}
	c.Set(v)
	fmt.Print(c)

	// Output:
	// 58
}

func ExampleCounter_SetFromText() {

	metValue := "0.01"

	predVal, _ := strconv.ParseFloat(metValue, 64)
	g := repository.Gauge(predVal)

	fmt.Print(g)

	// Output:
	// 0.01
}
