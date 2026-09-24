package tiktok

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

// videoPathRe matches /@author/video/<digits> (optional trailing query/fragment).
var videoPathRe = regexp.MustCompile(`^/@([^/]+)/video/(\d+)`)

// ParseVideoURL normalizes a TikTok video URL to canonical form and extracts
// videoID + author. Query params never participate in identity.
func ParseVideoURL(raw string) (canonical, videoID, author string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", "", false
	}
	host := strings.ToLower(u.Host)
	if !strings.Contains(host, "tiktok.com") {
		return "", "", "", false
	}
	m := videoPathRe.FindStringSubmatch(u.Path)
	if m == nil {
		return "", "", "", false
	}
	author, videoID = m[1], m[2]
	canonical = "https://www.tiktok.com/@" + author + "/video/" + videoID
	return canonical, videoID, author, true
}

// NormalizeURL returns the canonical URL or "" if not a video URL.
func NormalizeURL(raw string) string {
	c, _, _, ok := ParseVideoURL(raw)
	if !ok {
		return ""
	}
	return c
}

// rawHit is one DOM anchor harvested by ScanJS.
type rawHit struct {
	URL    string `json:"url"`
	Author string `json:"author"`
	Title  string `json:"title"`
}

// BuildCandidates converts raw DOM hits into deduplicated VideoCandidates.
// First layer of dedup: key = VideoID.
func BuildCandidates(hits []rawHit, keyword string) []VideoCandidate {
	seen := map[string]bool{}
	out := []VideoCandidate{}
	for _, h := range hits {
		canon, vid, author, ok := ParseVideoURL(h.URL)
		if !ok {
			continue
		}
		if seen[vid] {
			continue
		}
		seen[vid] = true
		name := strings.TrimSpace(h.Author)
		if name == "" {
			name = author
		}
		if !strings.HasPrefix(name, "@") {
			name = "@" + name
		}
		out = append(out, VideoCandidate{
			VideoID:    vid,
			URL:        canon,
			AuthorID:   author,
			AuthorName: name,
			Title:      strings.TrimSpace(h.Title),
			Source:     "search-dom",
			Keyword:    keyword,
		})
	}
	return out
}

// ParseScanResult decodes the JSON array returned by ScanJS.
func ParseScanResult(jsonArr string, keyword string) ([]VideoCandidate, error) {
	var hits []rawHit
	if err := json.Unmarshal([]byte(jsonArr), &hits); err != nil {
		return nil, err
	}
	return BuildCandidates(hits, keyword), nil
}

// MergeDedup merges new candidates into acc keyed by VideoID.
// Returns (added, duplicates).
func MergeDedup(acc map[string]VideoCandidate, batch []VideoCandidate) (added, duplicates int) {
	for _, c := range batch {
		if _, ok := acc[c.VideoID]; ok {
			duplicates++
			continue
		}
		acc[c.VideoID] = c
		added++
	}
	return added, duplicates
}

// ScanJS harvests video anchors from the search DOM.
// Rule: a[href*="/video/"] with /@user/video/<id>; author link + accessible
// name as auxiliary sources; title best-effort, never fatal.
const ScanJS = `(function(){var out=[];var as=document.querySelectorAll('a[href*="/video/"]');for(var i=0;i<as.length;i++){var a=as[i];var href=a.getAttribute('href')||'';if(href.charAt(0)==='/'){href='https://www.tiktok.com'+href}if(href.indexOf('/video/')<0){continue}var author='';var m=href.match(/\/@([^\/\?]+)\/video\//);if(m){author='@'+m[1]}var t=a.getAttribute('aria-label')||a.getAttribute('title')||(a.innerText||'').trim().slice(0,200);out.push({url:href.split('?')[0].split('#')[0],author:author,title:t})}return out})()`

// ScrollJS scrolls the real scroll container (not blind window.scrollTo):
// prefers the deepest scrollable ancestor of the search results, else
// document.scrollingElement. Returns {scrolled, top, height}.
const ScrollJS = `(function(){function scrollable(el){if(!el||el===document.body||el===document.documentElement){return false}var s=getComputedStyle(el);return (s.overflowY==='auto'||s.overflowY==='scroll')&&el.scrollHeight>el.clientHeight+50}var anchor=document.querySelector('a[href*="/video/"]');var el=anchor?anchor.parentElement:null;var target=null;while(el){if(scrollable(el)){target=el}el=el.parentElement}if(!target){target=document.scrollingElement||document.documentElement}var before=target.scrollTop;target.scrollTop=target.scrollHeight;return {scrolled:true,top:before,height:target.scrollHeight}})()`

// HeightJS reports current scroll height (stability check).
const HeightJS = `(function(){var anchor=document.querySelector('a[href*="/video/"]');var el=anchor?anchor.parentElement:null;var target=null;function scrollable(e){if(!e||e===document.body||e===document.documentElement){return false}var s=getComputedStyle(e);return (s.overflowY==='auto'||s.overflowY==='scroll')&&e.scrollHeight>e.clientHeight+50}while(el){if(scrollable(el)){target=el}el=el.parentElement}if(!target){target=document.scrollingElement||document.documentElement}return {height:target.scrollHeight,top:target.scrollTop}})()`
