package jcs_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ucarion/jcs"
)

func ExampleFormat() {
	input := `{"z": [1, 2, 3], "a": "<foo>" }`

	var v interface{}
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		panic(err)
	}

	// This is not going to output JCS canonical JSON, because of quirks in
	// encoding/json's output format.
	outNaive, _ := json.Marshal(v)
	fmt.Println(string(outNaive))

	out, _ := jcs.Format(v)
	fmt.Println(out)
	// Output:
	// {"a":"\u003cfoo\u003e","z":[1,2,3]}
	// {"a":"<foo>","z":[1,2,3]}
}

func TestAppend(t *testing.T) {
	b := []byte{'x', 'y', 'z'}
	b, err := jcs.Append(b, []interface{}{"foo"})
	assert.NoError(t, err)
	assert.Equal(t, []byte{'x', 'y', 'z', '[', '"', 'f', 'o', 'o', '"', ']'}, b)
}

func TestUnsupportedType(t *testing.T) {
	// Note: this is a map[string]string, instead of the required
	// map[string]interface{}.
	_, err := jcs.Format(map[string]string{"foo": "bar"})
	assert.Equal(t, err, jcs.ErrUnsupportedType)
}

func TestFormat(t *testing.T) {
	files, err := ioutil.ReadDir("testdata/input")
	assert.NoError(t, err)

	for _, file := range files {
		t.Run(file.Name(), func(t *testing.T) {
			inData, err := ioutil.ReadFile(filepath.Join("testdata/input", file.Name()))
			assert.NoError(t, err)

			var in interface{}
			assert.NoError(t, json.Unmarshal(inData, &in))

			out, err := ioutil.ReadFile(filepath.Join("testdata/output", file.Name()))
			assert.NoError(t, err)

			actual, err := jcs.Format(in)
			assert.NoError(t, err)

			assert.Equal(t, out, []byte(actual))
		})
	}
}

func TestFormatFloat(t *testing.T) {
	testCases := []struct {
		in  string
		out string
		err error
	}{
		{in: "0000000000000000", out: "0"},
		{in: "8000000000000000", out: "0"},
		{in: "7fefffffffffffff", out: "1.7976931348623157e+308"},
		{in: "ffefffffffffffff", out: "-1.7976931348623157e+308"},
		{in: "4340000000000000", out: "9007199254740992"},
		{in: "c340000000000000", out: "-9007199254740992"},
		{in: "4430000000000000", out: "295147905179352830000"},
		{in: "7fffffffffffffff", out: "", err: jcs.ErrNaN},
		{in: "7ff0000000000000", out: "", err: jcs.ErrInf},
		{in: "44b52d02c7e14af5", out: "9.999999999999997e+22"},
		{in: "44b52d02c7e14af6", out: "1e+23"},
		{in: "44b52d02c7e14af7", out: "1.0000000000000001e+23"},
		{in: "444b1ae4d6e2ef4e", out: "999999999999999700000"},
		{in: "444b1ae4d6e2ef4f", out: "999999999999999900000"},
		{in: "444b1ae4d6e2ef50", out: "1e+21"},
		{in: "3eb0c6f7a0b5ed8c", out: "9.999999999999997e-7"},
		{in: "3eb0c6f7a0b5ed8d", out: "0.000001"},
		{in: "41b3de4355555553", out: "333333333.3333332"},
		{in: "41b3de4355555554", out: "333333333.33333325"},
		{in: "41b3de4355555555", out: "333333333.3333333"},
		{in: "41b3de4355555556", out: "333333333.3333334"},
		{in: "41b3de4355555557", out: "333333333.33333343"},
		{in: "becbf647612f3696", out: "-0.0000033333333333333333"},
		{in: "43143ff3c1cb0959", out: "1424953923781206.2"},
	}

	for _, tt := range testCases {
		t.Run(tt.in, func(t *testing.T) {
			testFloatC14N(t, tt.in, tt.out, tt.err)
		})
	}
}

func TestFormatFloat100M(t *testing.T) {
	if os.Getenv("JCS_TEST_100M") != "1" {
		t.Skip("JCS_TEST_100M not set to 1")
	}

	f, err := os.Open("es6testfile100m.txt")
	assert.NoError(t, err)

	defer f.Close()

	i := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		i++

		if i%1000000 == 0 {
			t.Logf("es6 float test progress indicator: %d", i)
		}

		line := scanner.Text()
		sep := strings.IndexByte(line, ',')

		testFloatC14N(t, line[:sep], line[sep+1:], nil)
	}
}

func testFloatC14N(t *testing.T, in string, out string, outError error) {
	inBits, err := strconv.ParseUint(in, 16, 64)
	assert.NoError(t, err)

	inFloat := math.Float64frombits(inBits)

	actual, actualErr := jcs.Format(inFloat)
	assert.Equal(t, outError, actualErr)
	assert.Equal(t, out, actual, "bad float for input: %v, want: %v, got: %v", in, out, actual)
}
