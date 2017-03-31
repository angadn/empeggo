// mpg123.go contains all bindings to the C library

package mpg123

/*
#include <mpg123.h>
#cgo CFLAGS: -I/usr/local/include
#cgo LDFLAGS: -L/usr/local/lib -lmpg123
*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"unsafe"
)

var EOF = errors.New("EOF")

// All output encoding formats supported by mpg123
const (
	ENC_8           = C.MPG123_ENC_8
	ENC_16          = C.MPG123_ENC_16
	ENC_24          = C.MPG123_ENC_24
	ENC_32          = C.MPG123_ENC_32
	ENC_SIGNED      = C.MPG123_ENC_SIGNED
	ENC_FLOAT       = C.MPG123_ENC_FLOAT
	ENC_SIGNED_8    = C.MPG123_ENC_SIGNED_8
	ENC_UNSIGNED_8  = C.MPG123_ENC_UNSIGNED_8
	ENC_ULAW_8      = C.MPG123_ENC_ULAW_8
	ENC_ALAW_8      = C.MPG123_ENC_ALAW_8
	ENC_SIGNED_16   = C.MPG123_ENC_SIGNED_16
	ENC_UNSIGNED_16 = C.MPG123_ENC_UNSIGNED_16
	ENC_SIGNED_24   = C.MPG123_ENC_SIGNED_24
	ENC_UNSIGNED_24 = C.MPG123_ENC_UNSIGNED_24
	ENC_SIGNED_32   = C.MPG123_ENC_SIGNED_32
	ENC_UNSIGNED_32 = C.MPG123_ENC_UNSIGNED_32
	ENC_FLOAT_32    = C.MPG123_ENC_FLOAT_32
	ENC_FLOAT_64    = C.MPG123_ENC_FLOAT_64
	ENC_ANY         = C.MPG123_ENC_ANY
)

// Contains a handle for and mpg123 decoder instance
type Decoder struct {
	handle *C.mpg123_handle
}

// Initialize the mpg123 library when package is loaded
func init() {
	err := C.mpg123_init()
	if err != C.MPG123_OK {
		//return fmt.Errorf("error initializing mpg123")
		panic("failed to initialize mpg123")
	}
	//return nil
}

///////////////////////////
// DECODER INSTANCE CODE //
///////////////////////////

// Create a new mpg123 decoder instance
func NewDecoder(decoder string) (*Decoder, error) {
	var err C.int
	var mh *C.mpg123_handle
	if decoder != "" {
		mh = C.mpg123_new(nil, &err)
	} else {
		cdecoder := C.CString(decoder)
		defer C.free(unsafe.Pointer(cdecoder))
		mh = C.mpg123_new(cdecoder, &err)
	}
	if mh == nil {
		errstring := C.mpg123_plain_strerror(err)
		defer C.free(unsafe.Pointer(errstring))
		return nil, fmt.Errorf("Error initializing mpg123 decoder: %s", errstring)
	}
	dec := new(Decoder)
	dec.handle = mh
	return dec, nil
}

// Delete mpg123 decoder instance
func (d *Decoder) Delete() {
	C.mpg123_delete(d.handle)
}

// returns a string containing the most recent error message corresponding to
// an mpg123 decoder instance
func (d *Decoder) strerror() string {
	return C.GoString(C.mpg123_strerror(d.handle))
}

////////////////////////
// OUTPUT FORMAT CODE //
////////////////////////

// Disable all decoder output formats. Use before specifying supported formats
func (d *Decoder) FormatNone() {
	C.mpg123_format_none(d.handle)
}

// Enable all decoder output formats. This is the default setting.
func (d *Decoder) FormatAll() {
	C.mpg123_format_all(d.handle)
}

// Returns current output format
func (d *Decoder) GetFormat() (rate int64, channels int, encoding int) {
	var cRate C.long
	var cChans, cEnc C.int
	C.mpg123_getformat(d.handle, &cRate, &cChans, &cEnc)
	return int64(cRate), int(cChans), int(cEnc)
}

// Set the audio output format for decoder
func (d *Decoder) Format(rate int64, channels int, encodings int) {
	C.mpg123_format(d.handle, C.long(rate), C.int(channels), C.int(encodings))
}

/////////////////////////////
// INPUT AND DECODING CODE //
/////////////////////////////

// Open an mp3 file for decoding using a filename
func (d *Decoder) Open(file string) error {
	cfile := C.CString(file)
	defer C.free(unsafe.Pointer(cfile))
	err := C.mpg123_open(d.handle, cfile)
	if err != C.MPG123_OK {
		return fmt.Errorf("Error opening %s: %s\n", file, d.strerror())
	}
	return nil
}

// Bind an open *os.File for decoding
func (d *Decoder) OpenFile(f *os.File) error {
	err := C.mpg123_open_fd(d.handle, C.int(f.Fd()))
	if err != C.MPG123_OK {
		return fmt.Errorf("Error attaching file: %s", d.strerror())
	}
	return nil
}

// Prepare for direct feeding via Feed
func (d *Decoder) OpenFeed() error {
	err := C.mpg123_open_feed(d.handle)
	if err != C.MPG123_OK {
		return fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	return nil
}

// Close input file if one was opened by mpg123
func (d *Decoder) Close() error {
	err := C.mpg123_close(d.handle)
	if err != C.MPG123_OK {
		return fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	return nil
}

// Read data from stream and decode into buf. Returns number of bytes decoded.
func (d *Decoder) Read(buf []byte) (int, error) {
	var done C.size_t
	err := C.mpg123_read(d.handle, (*C.uchar)(&buf[0]), C.size_t(len(buf)), &done)
	if err == C.MPG123_DONE {
		return int(done), EOF
	}
	if err != C.MPG123_OK {
		return int(done), fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	return int(done), nil
}

// Feed bytes into the decoder
func (d *Decoder) Feed(buf []byte) error {
	err := C.mpg123_feed(d.handle, (*C.uchar)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
	if err != C.MPG123_OK {
		return fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	return nil
}

// DecoderReader is the way to decode streaming MP3
type DecoderReader struct {
	decoder  *Decoder
	src      io.Reader
	fps      int
	channels int
	paranoid bool
}

// Paranoid mode shuts off the decoder on a non-EOF error (handy if your input is a duplex network stream).
func (dr *DecoderReader) Paranoid() *DecoderReader {
	dr.paranoid = true
	return dr
}

// Nuke kills our DecoderReader appropriately
func (dr DecoderReader) Nuke() {
	dr.decoder.Close()
	dr.decoder.Delete()
}

// Read duck-types DecoderReader into io.Reader.
func (dr DecoderReader) Read(bytes []byte) (int, error) {
	buf := make([]byte, 64*1024)
	for {
		var n int
		var err error

		// Feed data
		if n, err = dr.src.Read(buf); err == nil {
			if err = dr.decoder.Feed(buf[0:n]); err != nil {
				log.Print("Error while feeding to mpg123: ", err)
			}
		} else if err != io.EOF { // EOF in Feed does NOT mean EOF in Read!
			if dr.paranoid {
				dr.Nuke()
			}

			return 0, err
		} else if dr.paranoid {
			dr.Nuke()
			return 0, io.EOF
		}

		// Read output
		var done C.size_t
		msg := C.mpg123_read(dr.decoder.handle, (*C.uchar)(&bytes[0]), C.size_t(len(bytes)), &done)
		switch msg {
		case C.MPG123_NEW_FORMAT:
			rate, channel, encoding := dr.decoder.GetFormat()
			log.Printf(
				"New format with rate: %d, channels: %d, encoding: %d", rate, channel, encoding,
			)
			fallthrough
		case C.MPG123_OK:
			fallthrough
		case C.MPG123_DONE:
			fallthrough
		case C.MPG123_NEED_MORE:
			if done > 0 {
				return int(done), nil
			}
			if err == io.EOF {
				// Source exhausted, so signal EOF
				dr.Nuke()
				return int(done), io.EOF
			}
		}
	}
}

// DecoderReader gives you an io.Reader for streaming-decoding. It performs
// a combination of Feed and Read, and relies on you to first call OpenFeed
// before invoking DecoderReader.Read.
func (d *Decoder) DecoderReader(
	src io.Reader, fps int, channels int, encoding int,
) DecoderReader {
	d.FormatNone()
	d.Format(int64(fps), channels, encoding)
	return DecoderReader{
		decoder:  d,
		src:      src,
		fps:      fps,
		channels: channels,
		paranoid: false,
	}
}

// MonoDecoderReader is an alias that gives you an io.Reader for
// decoding a stream that is known to be mono-channeled.
func (d *Decoder) MonoDecoderReader(src io.Reader, fps int, encoding int) DecoderReader {
	return d.DecoderReader(src, fps, 1, encoding)
}
