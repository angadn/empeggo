Introduction
------------

empeggo is a fork to decode buffers with MPG123
This library is still very much a work in progress.

Usage
-----
#### Decoding a file
The mpg123 library is accessed via a Decoder struct containing a C pointer
to an instance of the library. To decode a file, first create an instance of the decoder,
then tell it to open the file.

	decoder, err := mpg123.NewDecoder("")
	err = decoder.Open("test.mp3")

At this point you should have the decoder peek into the file and find
the format it is encoded in. You may also want to lock this format in
as it may change later if you do not do so.

	rate, channels, encoding := decoder.GetFormat()
	// clear list of formats and only allow the current settings
	decoder.FormatNone()
	decoder.Format(rate, channels, encoding)

Now you are ready to start decoding the file. Simply create a buffer 
and read data into it. Note that there may still be data in the buffer
when EOF is returned, so check for errors after processing the buffer.

	buf := make([]byte, 1024*16)
	for {
		len, err := decoder.Read(buf)
		// do something with the PCM data
		if err != nil {
			break
		}
	}

#### Decoding a Reader
Useful when working with custom audio-protocols!

	decoder.OpenFeed() // You must call this manually first
	// Get a DecoderReader for an output format
	outputReader := decoder.DecoderReader(inputReader, 44100, 1, mpg123.ENC_SIGNED_16)

	buf := make([]byte, 16*1024)
	for {
		if n, err := outputReader.Read(buf); err != io.EOF {
			// Do your thing with buf[0:n]
		} else {
			break
		}
	}
	// outputReader will Close and Delete itself automatically when data is over 😇

Examples
--------

An example program is included in examples/mp3dump. This program decodes
an MP3 file and writes the raw PCM data to a file.

	go get github.com/bobertlo/go-mpg123/examples/mp3dump
	mp3dump <file.mp3> <outfile.raw>

This raw audio file may be played using mplayer:

	mplayer -demuxer rawaudio -rawaudio rate=<samplerate>:channels=<channels> out.raw
