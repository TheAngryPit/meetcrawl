package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/detect"
)

func (c *LiveClient) collectDocsUnderRoots(ctx context.Context, folderRoots []string) (map[string]DriveDoc, error) {
	roots, err := c.resolveFolderRoots(ctx, folderRoots)
	if err != nil {
		return nil, err
	}
	out := map[string]DriveDoc{}
	for _, rootID := range roots {
		if err := c.walkFolder(ctx, rootID, "", out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *LiveClient) resolveFolderRoots(ctx context.Context, folderRoots []string) ([]string, error) {
	if len(folderRoots) == 0 {
		folderRoots = []string{"Google Meet"}
	}
	var ids []string
	for _, root := range folderRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if detect.LooksLikeFolderID(root) {
			ids = append(ids, root)
			continue
		}
		id, err := c.findFolderByPath(ctx, root)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("drive: no meet_folder_roots resolved")
	}
	return ids, nil
}

func (c *LiveClient) findFolderByPath(ctx context.Context, path string) (string, error) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	parent := "root"
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		q := fmt.Sprintf("mimeType='application/vnd.google-apps.folder' and trashed=false and name='%s' and '%s' in parents",
			escapeDriveQuery(segment), parent)
		res, err := c.drive.Files.List().
			Q(q).
			Fields("files(id)").
			PageSize(1).
			Context(ctx).
			Do()
		if err != nil {
			return "", fmt.Errorf("drive folder lookup %q: %w", segment, err)
		}
		if len(res.Files) == 0 {
			return "", fmt.Errorf("drive folder %q not found under parent %q", segment, parent)
		}
		parent = res.Files[0].Id
	}
	return parent, nil
}

func escapeDriveQuery(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

func (c *LiveClient) walkFolder(ctx context.Context, folderID, path string, out map[string]DriveDoc) error {
	pageToken := ""
	for {
		call := c.drive.Files.List().
			Q(fmt.Sprintf("'%s' in parents and trashed=false", folderID)).
			Fields("nextPageToken, files(id, name, mimeType, modifiedTime, version)").
			PageSize(100).
			Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		res, err := call.Do()
		if err != nil {
			return fmt.Errorf("drive files.list: %w", err)
		}
		for _, f := range res.Files {
			childPath := path
			if childPath == "" {
				childPath = f.Name
			} else {
				childPath = childPath + "/" + f.Name
			}
			if f.MimeType == "application/vnd.google-apps.folder" {
				if err := c.walkFolder(ctx, f.Id, childPath, out); err != nil {
					return err
				}
				continue
			}
			if f.MimeType != "application/vnd.google-apps.document" {
				continue
			}
			if !detect.IsGeminiDocTitle(f.Name) {
				continue
			}
			mod, _ := parseTime(f.ModifiedTime)
			out[f.Id] = DriveDoc{
				ID:           f.Id,
				Name:         f.Name,
				MimeType:     f.MimeType,
				ModifiedTime: mod,
				RevisionID:   fmt.Sprintf("%d", f.Version),
				ParentPath:   path,
			}
		}
		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return nil
}
