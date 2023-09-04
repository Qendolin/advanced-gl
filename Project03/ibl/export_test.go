package ibl

// these functions are only exported when running tests

var EncodeRgbeChunk = encodeRgbeChunk
var DecodeRgbeChunk = decodeRgbeChunk

var SampleSequenceImplementation = &sampleSequenceImplementation
var HammersleySequence = generateHammersleySequence
var RobertsSequence = generateRobertsSequence
var RandomSequence = generateRandomSequence
