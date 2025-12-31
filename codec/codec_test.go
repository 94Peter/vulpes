// Package codec_test contains the unit tests for the codec package.
// It verifies the functionality of GOB and MessagePack codecs, error handling,
// and the global configuration options.
package codec

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
)

type testStruct struct {
	Name string
	Age  int
}

// resetDefaultCodec is a helper to reset the global state for tests.
func resetDefaultCodec() {
	defaultCodeMethod = GOB
	once = sync.Once{}
}

func TestGobCodec(t *testing.T) {
	codec := &gobCodec[testStruct]{}
	data := testStruct{Name: "test", Age: 10}

	encoded, err := codec.Encode(data)
	require.NoError(t, err)

	decoded, err := codec.Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
	assert.Equal(t, GOB, codec.Method())
}

func TestMsgPackCodec(t *testing.T) {
	codec := &msgpackCodec[testStruct]{}
	data := testStruct{Name: "test", Age: 10}

	encoded, err := codec.Encode(data)
	require.NoError(t, err)

	decoded, err := codec.Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
	assert.Equal(t, MSGPACK, codec.Method())
}

func TestEncodeDecode(t *testing.T) {
	originalMethod := defaultCodeMethod
	defer func() {
		defaultCodeMethod = originalMethod
	}()

	data := testStruct{Name: "test", Age: 10}

	t.Run("GOB", func(t *testing.T) {
		defaultCodeMethod = GOB
		encoded, err := Encode(data)
		require.NoError(t, err)
		decoded, err := Decode[testStruct](encoded)
		require.NoError(t, err)
		assert.Equal(t, data, decoded)
	})

	t.Run("MSGPACK", func(t *testing.T) {
		defaultCodeMethod = MSGPACK
		encoded, err := Encode(data)
		require.NoError(t, err)
		decoded, err := Decode[testStruct](encoded)
		require.NoError(t, err)
		assert.Equal(t, data, decoded)
	})
}

func TestDecodeError(t *testing.T) {
	originalMethod := defaultCodeMethod
	defer func() {
		defaultCodeMethod = originalMethod
	}()

	t.Run("GOB", func(t *testing.T) {
		defaultCodeMethod = GOB
		t.Run("InvalidBase64", func(t *testing.T) {
			_, err := Decode[testStruct]("invalid base64")
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrBase64DecodeFailed)
		})

		t.Run("InvalidGob", func(t *testing.T) {
			// "invalid gob" base64 encoded
			_, err := Decode[testStruct]("aW52YWxpZCBnb2I=")
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrGobDecodeFailed)
		})
	})

	t.Run("MSGPACK", func(t *testing.T) {
		defaultCodeMethod = MSGPACK
		t.Run("InvalidBase64", func(t *testing.T) {
			_, err := Decode[testStruct]("invalid base64")
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrBase64DecodeFailed)
		})

		t.Run("InvalidMsgPack", func(t *testing.T) {
			// "invalid msgpack" base64 encoded
			_, err := Decode[testStruct]("aW52YWxpZCBtc2dwYWNr")
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrMsgPackDecodeFailed)
		})
	})
}

func TestWithCodecMethod(t *testing.T) {
	resetDefaultCodec()
	defer resetDefaultCodec()

	assert.Equal(t, GOB, defaultCodeMethod)

	// First call should set the method
	WithCodecMethod(MSGPACK)
	assert.Equal(t, MSGPACK, defaultCodeMethod)

	// Second call should not change it
	WithCodecMethod(GOB)
	assert.Equal(t, MSGPACK, defaultCodeMethod)
}

func TestUnknownCodecMethod(t *testing.T) {
	originalMethod := defaultCodeMethod
	defer func() {
		defaultCodeMethod = originalMethod
	}()

	defaultCodeMethod = "unknown"

	_, err := Encode(testStruct{Name: "test", Age: 10})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnknownCodecMethod)

	_, err = Decode[testStruct]("some string")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownCodecMethod)
}

func TestToStatus(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		var err error = nil
		st := ToStatus(err)
		assert.Nil(t, st)
	})

	t.Run("non-nil error", func(t *testing.T) {
		originalErr := errors.New("original error")
		msg := "error message"
		err := fmt.Errorf("%w: %s", originalErr, msg)

		st := ToStatus(err)
		assert.NotNil(t, st)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Equal(t, "codec error", st.Message())

		details := st.Details()
		assert.Len(t, details, 1)

		precond, ok := details[0].(*errdetails.PreconditionFailure)
		assert.True(t, ok, "expected PreconditionFailure detail, got %T", details[0])
		if ok {
			assert.Len(t, precond.Violations, 1)
			violation := precond.Violations[0]
			assert.Equal(t, "CODEC", violation.Type)
			assert.Equal(t, originalErr.Error(), violation.Subject)
			assert.Equal(t, err.Error(), violation.Description)
		}
	})
}
