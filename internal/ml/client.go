package ml

import (
	"context"
	"time"

	pb "github.com/example/youtube-dialogue-crawler/internal/ml/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.MLServiceClient
}

func NewClient(addr string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewMLServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

type FaceDetectionResult struct {
	AvgFaceCount float64
	MaxFaceCount int
	MinFaceCount int
	FrameResults []FrameFaceResult
}

type FrameFaceResult struct {
	Timestamp int
	FaceCount int
}

func (c *Client) DetectFaces(ctx context.Context, videoPath string) (*FaceDetectionResult, error) {
	req := &pb.FaceDetectionRequest{
		VideoPath:   videoPath,
		SampleRate:  5, // Sample every 5 seconds
	}

	resp, err := c.client.DetectFaces(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &FaceDetectionResult{
		AvgFaceCount: resp.AvgFaceCount,
		MaxFaceCount: int(resp.MaxFaceCount),
		MinFaceCount: int(resp.MinFaceCount),
	}

	for _, fr := range resp.FrameResults {
		result.FrameResults = append(result.FrameResults, FrameFaceResult{
			Timestamp: int(fr.TimestampMs),
			FaceCount: int(fr.FaceCount),
		})
	}

	return result, nil
}

type SpeakerAnalysisResult struct {
	SpeakerCount   int
	DialogueRatio  float64
	Segments       []SpeakerSegment
}

type SpeakerSegment struct {
	StartMs   int
	EndMs     int
	SpeakerID int
}

func (c *Client) AnalyzeSpeakers(ctx context.Context, videoPath string) (*SpeakerAnalysisResult, error) {
	req := &pb.SpeakerAnalysisRequest{
		VideoPath: videoPath,
	}

	resp, err := c.client.AnalyzeSpeakers(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &SpeakerAnalysisResult{
		SpeakerCount:  int(resp.SpeakerCount),
		DialogueRatio: resp.DialogueRatio,
	}

	for _, seg := range resp.Segments {
		result.Segments = append(result.Segments, SpeakerSegment{
			StartMs:   int(seg.StartMs),
			EndMs:     int(seg.EndMs),
			SpeakerID: int(seg.SpeakerId),
		})
	}

	return result, nil
}
