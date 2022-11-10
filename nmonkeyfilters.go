package main

import (
	"fmt"
	"io"
	"time"
)

type Filter interface {
	io.Reader
	SetSource(r io.Reader)
}

// =========================================================================================================
//
//	NullFilter
//
// =========================================================================================================
type NullFilter struct {
	src io.Reader
}

func (f *NullFilter) SetSource(r io.Reader) {
	f.src = r
}

func (f *NullFilter) Read(p []byte) (int, error) {
	return f.src.Read(p)
}

// =========================================================================================================
//		tapFilter
// =========================================================================================================

type TapFilter struct {
	NullFilter
	tapDest EndPoint
}

func MakeTapFilter(config string) (Filter, error) {
	if ep := GetEndPoint(config); ep != nil {
		return &TapFilter{tapDest: ep}, nil
	}

	return nil, fmt.Errorf("unknown Endpoint name for tap Filter destination: %v", config)
}

func (f *TapFilter) Read(p []byte) (int, error) {
	nr, err := f.src.Read(p)
	if err != nil {
		return nr, err
	}

	nw, err := f.tapDest.Write(p[:nr])
	if err != nil {
		return nw, err
	}

	return nr, err
}

// =========================================================================================================
//
//	ToHexFilter
//
// =========================================================================================================
type ToHexFilter struct {
	NullFilter
	input  []byte
	output []byte
	buffer []byte
}

const hexString = "0123456789abcdef"

func MakeToHexFilter() *ToHexFilter {
	return &ToHexFilter{input: make([]byte, 128), output: make([]byte, 256)}
}

func (f *ToHexFilter) Read(p []byte) (int, error) {

	if len(f.buffer) == 0 {
		n, err := f.src.Read(f.input)

		if err == nil {
			for i := 0; i < n; i++ {
				f.output[i*2] = hexString[f.input[i]>>4]
				f.output[i*2+1] = hexString[f.input[i]&0xf]
			}
			f.buffer = f.output[:n*2]
		} else {
			return 0, err
		}
	}

	if len(f.buffer) > 0 {
		n := copy(p, f.buffer)
		f.buffer = f.buffer[n:]
		return n, nil
	}

	return 0, nil
}

// =========================================================================================================
//		EatEOF Filter
// =========================================================================================================

type EatEOFFilter struct {
	NullFilter
}

func (f *EatEOFFilter) Read(p []byte) (int, error) {
	num, err := f.src.Read(p)
	if err == io.EOF {
		for {
			time.Sleep(time.Second)
		}
	}

	return num, err
}

// =========================================================================================================
//
//	Delay Filter: delay(delayMS, blockSize)
//
// =========================================================================================================
// http://play.golang.org/p/bLJ9mMrVHc
type DelayFilter struct {
	NullFilter
	delayMS int
	input   []byte
	buffer  []byte
}

func MakeDelayFilter(chunksize, delayMS int) Filter {
	return &DelayFilter{input: make([]byte, chunksize), delayMS: delayMS}
}

func (f *DelayFilter) Read(p []byte) (int, error) {
	if len(f.buffer) == 0 {
		time.Sleep(time.Duration(f.delayMS) * time.Millisecond)
		nr, err := f.src.Read(f.input)
		if err == nil {
			f.buffer = f.input[:nr]
		} else {
			return 0, err
		}
	}

	if len(f.buffer) > 0 {
		n := copy(p, f.buffer)
		f.buffer = f.buffer[n:]
		return n, nil
	}

	return 0, nil
}

// =========================================================================================================
//		Filter Factory
// =========================================================================================================

type FilterMaker func(config string) (Filter, error)

var FilterFactory = map[string]FilterMaker{

	"nullFilter": func(config string) (Filter, error) {
		return &NullFilter{}, nil
	},

	"toHex": func(config string) (Filter, error) {
		return MakeToHexFilter(), nil
	},

	"tap": func(config string) (Filter, error) {
		return MakeTapFilter(config)
	},

	"eatEOF": func(config string) (Filter, error) {
		return &EatEOFFilter{}, nil
	},

	"delay": func(config string) (Filter, error) {
		var chunksize, delayMS int
		_, err := fmt.Sscanf(config, "%d,%d", &chunksize, &delayMS)
		if err != nil {
			return nil, fmt.Errorf("error making delay(%s) filter %w", config, err)
		}
		return MakeDelayFilter(chunksize, delayMS), nil
	},
}
