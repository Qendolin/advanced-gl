package ibl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

func DecodeOldIblEnv(r io.Reader) (env *IblEnv, err error) {
	le := binary.LittleEndian

	header := iblEnvHeader1_001_000{}
	err = binary.Read(r, le, &header)
	if err != nil {
		return nil, err
	}

	if header.Check != MagicNumberIBLENV {
		return nil, fmt.Errorf("environment header is corrupt")
	}

	buf := new(bytes.Buffer)
	switch header.Version {
	case IblEnvVersion1_001_000:
		newHeader := iblEnvHeader1_002_000{
			iblEnvHeader1_001_000: header,
			Levels:                1,
		}
		newHeader.Version = IblEnvVersion1_002_000
		err = binary.Write(buf, le, newHeader)
		if err != nil {
			return nil, err
		}
		return DecodeOldIblEnv(io.MultiReader(buf, r))
	case IblEnvVersion1_002_000:
		binary.Write(buf, le, header)
		if err != nil {
			return nil, err
		}
		return DecodeIblEnv(io.MultiReader(buf, r))
	default:
		return nil, fmt.Errorf("environment version %d unsupported", header.Version)
	}
}
