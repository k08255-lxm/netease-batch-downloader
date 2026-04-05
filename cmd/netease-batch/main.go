package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/liuran001/MusicBot-Go/bot/download"
	"github.com/liuran001/MusicBot-Go/bot/id3"
	"github.com/liuran001/MusicBot-Go/bot/platform"
	"github.com/liuran001/MusicBot-Go/plugins/netease"
	"gopkg.in/ini.v1"
)

type appConfig struct {
	MusicU               string
	SpoofIP              bool
	DownloadTimeout      time.Duration
	DownloadProxy        string
	CheckMD5             bool
	EnableMultipart      bool
	MultipartConcurrency int
	MultipartMinSizeMB   int64
}

type manifest struct {
	GeneratedAt       time.Time     `json:"generated_at"`
	SourceURL         string        `json:"source_url"`
	Quality           string        `json:"quality"`
	OutputDir         string        `json:"output_dir"`
	PlaylistID        string        `json:"playlist_id"`
	PlaylistTitle     string        `json:"playlist_title"`
	PlaylistCreator   string        `json:"playlist_creator,omitempty"`
	PlaylistCoverURL  string        `json:"playlist_cover_url,omitempty"`
	PlaylistCoverFile string        `json:"playlist_cover_file,omitempty"`
	TrackCount        int           `json:"track_count"`
	Results           []trackResult `json:"results"`
}

type trackResult struct {
	Index          int      `json:"index"`
	TrackID        string   `json:"track_id"`
	Title          string   `json:"title"`
	Artists        []string `json:"artists"`
	Album          string   `json:"album,omitempty"`
	Status         string   `json:"status"`
	File           string   `json:"file,omitempty"`
	CoverFile      string   `json:"cover_file,omitempty"`
	LyricsFile     string   `json:"lyrics_file,omitempty"`
	MetadataFile   string   `json:"metadata_file,omitempty"`
	Quality        string   `json:"quality,omitempty"`
	Format         string   `json:"format,omitempty"`
	Bitrate        int      `json:"bitrate,omitempty"`
	Size           int64    `json:"size,omitempty"`
	TrackURL       string   `json:"track_url,omitempty"`
	CoverURL       string   `json:"cover_url,omitempty"`
	Downloaded     bool     `json:"downloaded"`
	Skipped        bool     `json:"skipped"`
	Warnings       []string `json:"warnings,omitempty"`
	Error          string   `json:"error,omitempty"`
	DurationSecond int64    `json:"duration_second,omitempty"`
}

type trackSidecar struct {
	GeneratedAt     time.Time `json:"generated_at"`
	SourcePlaylist  string    `json:"source_playlist"`
	PlaylistTitle   string    `json:"playlist_title"`
	PlaylistIndex   int       `json:"playlist_index"`
	TrackID         string    `json:"track_id"`
	TrackURL        string    `json:"track_url,omitempty"`
	Title           string    `json:"title"`
	Artists         []string  `json:"artists"`
	Album           string    `json:"album,omitempty"`
	AlbumArtists    []string  `json:"album_artists,omitempty"`
	CoverURL        string    `json:"cover_url,omitempty"`
	CoverFile       string    `json:"cover_file,omitempty"`
	LyricsFile      string    `json:"lyrics_file,omitempty"`
	OutputFile      string    `json:"output_file"`
	DurationSecond  int64     `json:"duration_second,omitempty"`
	TrackNumber     int       `json:"track_number,omitempty"`
	DiscNumber      int       `json:"disc_number,omitempty"`
	Year            int       `json:"year,omitempty"`
	Quality         string    `json:"quality,omitempty"`
	Format          string    `json:"format,omitempty"`
	Bitrate         int       `json:"bitrate,omitempty"`
	Size            int64     `json:"size,omitempty"`
	MD5             string    `json:"md5,omitempty"`
	DownloadURL     string    `json:"download_url,omitempty"`
	DownloadedAt    time.Time `json:"downloaded_at"`
	EmbeddedArtwork bool      `json:"embedded_artwork"`
	EmbeddedLyrics  bool      `json:"embedded_lyrics"`
}

type job struct {
	Index int
	Track platform.Track
}

func main() {
	var (
		configPath  string
		rawURL      string
		outDir      string
		musicU      string
		qualityText string
		concurrency int
		lyrics      bool
		covers      bool
		overwrite   bool
		checkOnly   bool
	)

	flag.StringVar(&configPath, "config", "config.ini", "配置文件路径，会读取 [plugins.netease] music_u")
	flag.StringVar(&rawURL, "url", "", "网易云歌单或专辑 URL")
	flag.StringVar(&outDir, "out", "downloads", "输出根目录")
	flag.StringVar(&musicU, "music-u", "", "直接指定网易云 MUSIC_U，优先级高于配置文件")
	flag.StringVar(&qualityText, "quality", "lossless", "音质: standard/high/lossless/hires")
	flag.IntVar(&concurrency, "concurrency", 4, "同时下载歌曲数")
	flag.BoolVar(&lyrics, "lyrics", true, "导出歌词 sidecar，并尝试写入音频标签")
	flag.BoolVar(&covers, "covers", true, "导出封面文件，并尝试写入音频标签")
	flag.BoolVar(&overwrite, "overwrite", false, "覆盖已存在文件")
	flag.BoolVar(&checkOnly, "check", false, "只校验网易云 cookie 是否可用于下载，不执行批量下载")
	flag.Parse()

	if !checkOnly && strings.TrimSpace(rawURL) == "" {
		fmt.Fprintln(os.Stderr, "缺少 -url 参数")
		flag.Usage()
		os.Exit(2)
	}

	quality, err := platform.ParseQuality(strings.ToLower(strings.TrimSpace(qualityText)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "无效音质: %v\n", err)
		os.Exit(2)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取配置失败: %v\n", err)
		os.Exit(1)
	}
	if trimmed := strings.TrimSpace(musicU); trimmed != "" {
		cfg.MusicU = trimmed
	}
	if strings.TrimSpace(cfg.MusicU) == "" {
		fmt.Fprintln(os.Stderr, "未找到网易云 MUSIC_U，请在 config.ini 的 [plugins.netease] music_u 填入，或使用 -music-u")
		os.Exit(1)
	}
	if concurrency <= 0 {
		concurrency = 1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client := netease.New(cfg.MusicU, cfg.SpoofIP, nil)
	plat := netease.NewPlatform(client, false)

	if checkOnly {
		checkCtx, checkCancel := context.WithTimeout(ctx, 30*time.Second)
		defer checkCancel()
		result, err := plat.CheckCookie(checkCtx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "校验失败: %v\n", err)
			os.Exit(1)
		}
		if !result.OK {
			fmt.Fprintf(os.Stderr, "Cookie 不可用: %s\n", strings.TrimSpace(result.Message))
			os.Exit(1)
		}
		fmt.Printf("Cookie 可用: %s\n", strings.TrimSpace(result.Message))
		return
	}

	playlistID, ok := plat.MatchPlaylistURL(rawURL)
	if !ok {
		fmt.Fprintln(os.Stderr, "URL 不是可识别的网易云歌单或专辑链接")
		os.Exit(1)
	}

	metaCtx, metaCancel := context.WithTimeout(ctx, 90*time.Second)
	defer metaCancel()

	pl, err := plat.GetPlaylist(metaCtx, playlistID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取歌单失败: %v\n", err)
		os.Exit(1)
	}
	if pl == nil || len(pl.Tracks) == 0 {
		fmt.Fprintln(os.Stderr, "歌单为空或读取不到歌曲")
		os.Exit(1)
	}

	absOut, err := filepath.Abs(outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析输出目录失败: %v\n", err)
		os.Exit(1)
	}
	rootDir := filepath.Join(absOut, sanitizePathComponent(pl.Title))
	tracksDir := filepath.Join(rootDir, "tracks")
	coversDir := filepath.Join(rootDir, "covers")
	if err := os.MkdirAll(tracksDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
		os.Exit(1)
	}
	if covers {
		if err := os.MkdirAll(coversDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "创建封面目录失败: %v\n", err)
			os.Exit(1)
		}
	}

	downloader := download.NewDownloadService(download.DownloadServiceOptions{
		Timeout:              cfg.DownloadTimeout,
		Proxy:                cfg.DownloadProxy,
		CheckMD5:             cfg.CheckMD5,
		MaxRetries:           3,
		EnableMultipart:      cfg.EnableMultipart,
		MultipartConcurrency: cfg.MultipartConcurrency,
		MultipartMinSize:     cfg.MultipartMinSizeMB * 1024 * 1024,
	})
	id3Service := id3.NewID3Service(nil)
	id3Provider := netease.NewID3Provider(client)

	results := make([]trackResult, len(pl.Tracks))
	var printMu sync.Mutex
	printf := func(format string, args ...any) {
		printMu.Lock()
		defer printMu.Unlock()
		fmt.Printf(format, args...)
	}

	playlistCoverFile := ""
	if covers {
		playlistCoverURL := strings.TrimSpace(pl.CoverURL)
		if playlistCoverURL != "" {
			coverPath := filepath.Join(rootDir, "playlist-cover"+inferImageExt(playlistCoverURL))
			if err := downloadFile(ctx, downloader, playlistCoverURL, coverPath, false); err == nil {
				playlistCoverFile = relPath(rootDir, coverPath)
			}
		}
	}

	printf("开始下载: %s (%d 首)\n", pl.Title, len(pl.Tracks))
	printf("输出目录: %s\n", rootDir)
	printf("音质: %s, 并发: %d\n", quality.String(), concurrency)

	jobs := make(chan job)
	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				result := processTrack(ctx, processOptions{
					playlist:   pl,
					sourceURL:  rawURL,
					rootDir:    rootDir,
					tracksDir:  tracksDir,
					coversDir:  coversDir,
					quality:    quality,
					withLyrics: lyrics,
					withCovers: covers,
					overwrite:  overwrite,
					plat:       plat,
					downloader: downloader,
					id3Service: id3Service,
					id3Source:  id3Provider,
					printf:     printf,
				}, item.Index, item.Track)
				results[item.Index] = result
			}
		}()
	}

sendLoop:
	for i, track := range pl.Tracks {
		select {
		case <-ctx.Done():
			break sendLoop
		case jobs <- job{Index: i, Track: track}:
		}
	}
	close(jobs)
	wg.Wait()

	outManifest := manifest{
		GeneratedAt:       time.Now(),
		SourceURL:         rawURL,
		Quality:           quality.String(),
		OutputDir:         rootDir,
		PlaylistID:        pl.ID,
		PlaylistTitle:     pl.Title,
		PlaylistCreator:   pl.Creator,
		PlaylistCoverURL:  pl.CoverURL,
		PlaylistCoverFile: playlistCoverFile,
		TrackCount:        len(pl.Tracks),
		Results:           results,
	}
	manifestPath := filepath.Join(rootDir, "playlist.json")
	if err := writeJSON(manifestPath, outManifest); err != nil {
		fmt.Fprintf(os.Stderr, "写入歌单清单失败: %v\n", err)
		os.Exit(1)
	}

	var downloadedCount int
	var skippedCount int
	var failedCount int
	for _, result := range results {
		switch result.Status {
		case "downloaded":
			downloadedCount++
		case "skipped":
			skippedCount++
		default:
			failedCount++
		}
	}

	printf("完成: 成功 %d, 跳过 %d, 失败 %d\n", downloadedCount, skippedCount, failedCount)
	printf("歌单清单: %s\n", manifestPath)

	if ctx.Err() != nil {
		fmt.Fprintf(os.Stderr, "任务被中断: %v\n", ctx.Err())
		os.Exit(1)
	}
	if failedCount > 0 {
		os.Exit(1)
	}
}

type processOptions struct {
	playlist   *platform.Playlist
	sourceURL  string
	rootDir    string
	tracksDir  string
	coversDir  string
	quality    platform.Quality
	withLyrics bool
	withCovers bool
	overwrite  bool
	plat       *netease.NeteasePlatform
	downloader *download.DownloadService
	id3Service *id3.ID3Service
	id3Source  *netease.ID3Provider
	printf     func(format string, args ...any)
}

func processTrack(ctx context.Context, opts processOptions, index int, track platform.Track) trackResult {
	result := trackResult{
		Index:          index + 1,
		TrackID:        track.ID,
		Title:          track.Title,
		Artists:        artistNames(track.Artists),
		Album:          albumTitle(track),
		TrackURL:       strings.TrimSpace(track.URL),
		CoverURL:       strings.TrimSpace(trackCoverURL(track)),
		DurationSecond: int64(track.Duration.Seconds()),
		Status:         "failed",
	}

	baseName := buildTrackBaseName(index, len(opts.playlist.Tracks), track)

	infoCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	info, err := opts.plat.GetDownloadInfo(infoCtx, track.ID, opts.quality)
	cancel()
	if err != nil {
		result.Error = "获取下载链接失败: " + err.Error()
		opts.printf("[%d/%d] 失败: %s (%s)\n", index+1, len(opts.playlist.Tracks), baseName, result.Error)
		return result
	}

	ext := normalizeAudioExt(info.Format)
	audioPath := filepath.Join(opts.tracksDir, sanitizePathComponent(baseName)+"."+ext)
	result.File = relPath(opts.rootDir, audioPath)
	result.Format = ext
	result.Quality = opts.quality.String()
	result.Bitrate = info.Bitrate
	result.Size = info.Size

	exists := fileExists(audioPath)
	downloaded := false
	if !exists || opts.overwrite {
		if err := downloadTrack(ctx, opts.downloader, info, audioPath); err != nil {
			result.Error = "下载失败: " + err.Error()
			opts.printf("[%d/%d] 失败: %s (%s)\n", index+1, len(opts.playlist.Tracks), baseName, result.Error)
			return result
		}
		downloaded = true
	} else {
		result.Skipped = true
	}

	tagCtx, tagCancel := context.WithTimeout(ctx, 45*time.Second)
	tagData, tagErr := opts.id3Source.GetTagData(tagCtx, &track, info)
	tagCancel()
	if tagErr != nil {
		result.Warnings = append(result.Warnings, "读取标签元数据失败: "+tagErr.Error())
	}
	if tagData != nil && strings.TrimSpace(tagData.AlbumArtist) == "" {
		tagData.AlbumArtist = strings.Join(albumArtistNames(track), ", ")
	}

	coverFile := ""
	if opts.withCovers {
		coverURL := strings.TrimSpace(trackCoverURL(track))
		if coverURL == "" && tagData != nil {
			coverURL = strings.TrimSpace(tagData.CoverURL)
		}
		if coverURL != "" {
			coverName := sanitizePathComponent(track.ID)
			if coverName == "" {
				coverName = sanitizePathComponent(baseName)
			}
			coverPath := filepath.Join(opts.coversDir, coverName+inferImageExt(coverURL))
			if err := downloadFile(ctx, opts.downloader, coverURL, coverPath, false); err != nil {
				result.Warnings = append(result.Warnings, "下载封面失败: "+err.Error())
			} else {
				coverFile = relPath(opts.rootDir, coverPath)
				result.CoverFile = coverFile
			}
		}
	}

	lyricsFile := ""
	if opts.withLyrics {
		lyricsText := ""
		if tagData != nil {
			lyricsText = strings.TrimSpace(tagData.Lyrics)
		}
		if lyricsText == "" {
			lyricCtx, lyricCancel := context.WithTimeout(ctx, 45*time.Second)
			lyricData, lyricErr := opts.plat.GetLyrics(lyricCtx, track.ID)
			lyricCancel()
			if lyricErr != nil {
				result.Warnings = append(result.Warnings, "读取歌词失败: "+lyricErr.Error())
			} else if lyricData != nil {
				lyricsText = strings.TrimSpace(lyricData.Plain)
			}
		}
		if lyricsText != "" {
			lyricsPath := strings.TrimSuffix(audioPath, filepath.Ext(audioPath)) + ".lrc"
			lyricsText = platform.NormalizeLRCTimestamps(lyricsText)
			if err := writeText(lyricsPath, lyricsText); err != nil {
				result.Warnings = append(result.Warnings, "写入歌词文件失败: "+err.Error())
			} else {
				lyricsFile = relPath(opts.rootDir, lyricsPath)
				result.LyricsFile = lyricsFile
			}
		}
	}

	if tagData != nil {
		if opts.withLyrics && tagData.Lyrics == "" && lyricsFile != "" {
			absLyricsPath := filepath.Join(opts.rootDir, filepath.FromSlash(lyricsFile))
			if data, readErr := os.ReadFile(absLyricsPath); readErr == nil {
				tagData.Lyrics = string(data)
			}
		}
		coverAbsPath := ""
		if coverFile != "" {
			coverAbsPath = filepath.Join(opts.rootDir, filepath.FromSlash(coverFile))
		}
		if err := opts.id3Service.EmbedTags(audioPath, tagData, coverAbsPath); err != nil {
			result.Warnings = append(result.Warnings, "写入音频标签失败: "+err.Error())
		}
	}

	sidecar := trackSidecar{
		GeneratedAt:     time.Now(),
		SourcePlaylist:  opts.sourceURL,
		PlaylistTitle:   opts.playlist.Title,
		PlaylistIndex:   index + 1,
		TrackID:         track.ID,
		TrackURL:        strings.TrimSpace(track.URL),
		Title:           track.Title,
		Artists:         artistNames(track.Artists),
		Album:           albumTitle(track),
		AlbumArtists:    albumArtistNames(track),
		CoverURL:        strings.TrimSpace(trackCoverURL(track)),
		CoverFile:       coverFile,
		LyricsFile:      lyricsFile,
		OutputFile:      relPath(opts.rootDir, audioPath),
		DurationSecond:  int64(track.Duration.Seconds()),
		TrackNumber:     track.TrackNumber,
		DiscNumber:      track.DiscNumber,
		Year:            track.Year,
		Quality:         opts.quality.String(),
		Format:          ext,
		Bitrate:         info.Bitrate,
		Size:            info.Size,
		MD5:             strings.TrimSpace(info.MD5),
		DownloadURL:     strings.TrimSpace(info.URL),
		DownloadedAt:    time.Now(),
		EmbeddedArtwork: coverFile != "",
		EmbeddedLyrics:  lyricsFile != "",
	}
	sidecarPath := strings.TrimSuffix(audioPath, filepath.Ext(audioPath)) + ".json"
	if err := writeJSON(sidecarPath, sidecar); err != nil {
		result.Error = "写入元数据文件失败: " + err.Error()
		opts.printf("[%d/%d] 失败: %s (%s)\n", index+1, len(opts.playlist.Tracks), baseName, result.Error)
		return result
	}

	result.MetadataFile = relPath(opts.rootDir, sidecarPath)
	result.Downloaded = downloaded
	if result.Skipped {
		result.Status = "skipped"
		opts.printf("[%d/%d] 跳过: %s\n", index+1, len(opts.playlist.Tracks), baseName)
	} else {
		result.Status = "downloaded"
		opts.printf("[%d/%d] 完成: %s\n", index+1, len(opts.playlist.Tracks), baseName)
	}
	return result
}

func loadConfig(path string) (appConfig, error) {
	cfg := appConfig{
		SpoofIP:              true,
		DownloadTimeout:      60 * time.Second,
		CheckMD5:             true,
		EnableMultipart:      true,
		MultipartConcurrency: 4,
		MultipartMinSizeMB:   5,
	}
	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}

	file, err := ini.Load(path)
	if err != nil {
		return cfg, err
	}

	root := file.Section("")
	neteaseSection := file.Section("plugins.netease")

	if value := strings.TrimSpace(neteaseSection.Key("music_u").String()); value != "" {
		cfg.MusicU = value
	}
	if cfg.MusicU == "" {
		cfg.MusicU = strings.TrimSpace(root.Key("MUSIC_U").String())
	}
	if neteaseSection.HasKey("spoof_ip") {
		cfg.SpoofIP = neteaseSection.Key("spoof_ip").MustBool(cfg.SpoofIP)
	}
	if root.HasKey("DownloadTimeout") {
		timeoutSec := root.Key("DownloadTimeout").MustInt(int(cfg.DownloadTimeout.Seconds()))
		if timeoutSec > 0 {
			cfg.DownloadTimeout = time.Duration(timeoutSec) * time.Second
		}
	}
	if root.HasKey("CheckMD5") {
		cfg.CheckMD5 = root.Key("CheckMD5").MustBool(cfg.CheckMD5)
	}
	if root.HasKey("EnableMultipartDownload") {
		cfg.EnableMultipart = root.Key("EnableMultipartDownload").MustBool(cfg.EnableMultipart)
	}
	if root.HasKey("MultipartConcurrency") {
		value := root.Key("MultipartConcurrency").MustInt(cfg.MultipartConcurrency)
		if value > 0 {
			cfg.MultipartConcurrency = value
		}
	}
	if root.HasKey("MultipartMinSizeMB") {
		value := root.Key("MultipartMinSizeMB").MustInt64(cfg.MultipartMinSizeMB)
		if value > 0 {
			cfg.MultipartMinSizeMB = value
		}
	}
	cfg.DownloadProxy = strings.TrimSpace(root.Key("DownloadProxy").String())

	return cfg, nil
}

func downloadTrack(ctx context.Context, downloader *download.DownloadService, info *platform.DownloadInfo, audioPath string) error {
	if err := os.MkdirAll(filepath.Dir(audioPath), 0o755); err != nil {
		return err
	}
	if _, err := downloader.Download(ctx, info, audioPath, nil); err != nil {
		return err
	}
	stat, err := os.Stat(audioPath)
	if err != nil {
		return err
	}
	if stat.Size() <= 0 {
		return errors.New("下载后文件大小为 0")
	}
	return nil
}

func downloadFile(ctx context.Context, downloader *download.DownloadService, rawURL, dest string, overwrite bool) error {
	if strings.TrimSpace(rawURL) == "" {
		return errors.New("empty url")
	}
	if !overwrite && fileExists(dest) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	info := &platform.DownloadInfo{URL: rawURL}
	if _, err := downloader.Download(ctx, info, dest, nil); err != nil {
		return err
	}
	stat, err := os.Stat(dest)
	if err != nil {
		return err
	}
	if stat.Size() <= 0 {
		return errors.New("downloaded file is empty")
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeText(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func buildTrackBaseName(index, total int, track platform.Track) string {
	width := len(strconv.Itoa(total))
	return fmt.Sprintf("%0*d - %s - %s", width, index+1, strings.Join(artistNames(track.Artists), ", "), track.Title)
}

func artistNames(artists []platform.Artist) []string {
	result := make([]string, 0, len(artists))
	for _, artist := range artists {
		name := strings.TrimSpace(artist.Name)
		if name == "" {
			continue
		}
		result = append(result, name)
	}
	return result
}

func albumArtistNames(track platform.Track) []string {
	if track.Album != nil && len(track.Album.Artists) > 0 {
		names := artistNames(track.Album.Artists)
		if len(names) > 0 {
			return names
		}
	}
	return artistNames(track.Artists)
}

func albumTitle(track platform.Track) string {
	if track.Album == nil {
		return ""
	}
	return strings.TrimSpace(track.Album.Title)
}

func trackCoverURL(track platform.Track) string {
	if track.Album != nil {
		if cover := strings.TrimSpace(track.Album.CoverURL); cover != "" {
			return cover
		}
	}
	return strings.TrimSpace(track.CoverURL)
}

func normalizeAudioExt(format string) string {
	format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(format, ".")))
	switch format {
	case "mp3", "flac", "m4a", "mp4":
		return format
	case "aac":
		return "m4a"
	case "":
		return "mp3"
	default:
		return format
	}
}

func inferImageExt(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ".jpg"
	}
	if idx := strings.Index(rawURL, "?"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	ext := strings.ToLower(filepath.Ext(rawURL))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return ext
	default:
		return ".jpg"
	}
}

func relPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}

func fileExists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.Mode().IsRegular() && stat.Size() > 0
}

func sanitizePathComponent(value string) string {
	replacer := strings.NewReplacer(
		"/", " ",
		"\\", " ",
		":", " ",
		"*", " ",
		"?", " ",
		"\"", " ",
		"<", " ",
		">", " ",
		"|", " ",
	)
	value = strings.TrimSpace(replacer.Replace(value))
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, ". ")
	if value == "" {
		return "untitled"
	}
	if len([]rune(value)) > 140 {
		value = string([]rune(value)[:140])
		value = strings.TrimRight(value, ". ")
	}
	if isWindowsReservedName(value) {
		value = "_" + value
	}
	if value == "" {
		return "untitled"
	}
	return value
}

func isWindowsReservedName(value string) bool {
	if value == "" {
		return false
	}
	base := value
	if idx := strings.Index(base, "."); idx >= 0 {
		base = base[:idx]
	}
	base = strings.ToUpper(strings.TrimSpace(base))
	switch base {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}
