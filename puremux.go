package main

import (
	"fmt"
	"os"

	"github.com/Eyevinn/mp4ff/bits"
	"github.com/Eyevinn/mp4ff/mp4"
)

// mergeM4S combines Bilibili's single-fragment video and audio DASH files.
// It is intentionally a fallback for portable builds; ffmpeg is preferred when present.
func mergeM4S(videoPath, audioPath, outputPath string) error {
	paths := []string{videoPath, audioPath}
	trackIDs := []uint32{1, 2}
	files := make([]*mp4.File, 0, 2)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := mp4.DecodeFileSR(bits.NewFixedSliceReader(data))
		if err != nil {
			return fmt.Errorf("解析 %s: %w", path, err)
		}
		if file.Init == nil || len(file.Init.Moov.Traks) != 1 || len(file.Segments) == 0 {
			return fmt.Errorf("不支持的 DASH 文件结构")
		}
		files = append(files, file)
	}

	combinedInit := files[0].Init
	for i, file := range files {
		init := file.Init
		init.Moov.Trak.Tkhd.TrackID = trackIDs[i]
		if init.Moov.Mvex != nil && init.Moov.Mvex.Trex != nil {
			init.Moov.Mvex.Trex.TrackID = trackIDs[i]
		}
		if i == 1 {
			combinedInit.Moov.AddChild(init.Moov.Trak)
			if init.Moov.Mvex != nil && init.Moov.Mvex.Trex != nil {
				combinedInit.Moov.Mvex.AddChild(init.Moov.Mvex.Trex)
			}
		}
	}

	combinedMedia := mp4.NewMediaSegmentWithoutStyp()
	sequence := uint32(1)
	for i, file := range files {
		for _, segment := range file.Segments {
			for _, fragment := range segment.Fragments {
				if len(fragment.Moof.Trafs) != 1 {
					return fmt.Errorf("不支持的 DASH 分片结构")
				}
				fragment.Moof.Mfhd.SequenceNumber = sequence
				sequence++
				fragment.Moof.Trafs[0].Tfhd.TrackID = trackIDs[i]
				combinedMedia.AddFragment(fragment)
			}
		}
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()
	if err := combinedInit.Encode(out); err != nil {
		return err
	}
	if err := combinedMedia.Encode(out); err != nil {
		return err
	}
	return nil
}
