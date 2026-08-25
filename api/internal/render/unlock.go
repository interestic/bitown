package render

import (
	"github.com/interestic/bitown/internal/citycore"
)

// FolderUnlock gates map placement by city sector values (issue #79).
// Zero means no requirement for that sector.
type FolderUnlock struct {
	MinPop int `json:"min_pop,omitempty"`
	MinInd int `json:"min_ind,omitempty"`
	MinCom int `json:"min_com,omitempty"`
	MinEnv int `json:"min_env,omitempty"`
	MinSec int `json:"min_sec,omitempty"`
	MinTra int `json:"min_tra,omitempty"`
}

func (u FolderUnlock) satisfied(c *citycore.City) bool {
	if c == nil {
		return true
	}
	if u.MinPop > 0 && c.Pop.Int() < u.MinPop {
		return false
	}
	if u.MinInd > 0 && c.Ind.Int() < u.MinInd {
		return false
	}
	if u.MinCom > 0 && c.Com.Int() < u.MinCom {
		return false
	}
	if u.MinEnv > 0 && c.Env.Int() < u.MinEnv {
		return false
	}
	if u.MinSec > 0 && c.Sec.Int() < u.MinSec {
		return false
	}
	if u.MinTra > 0 && c.Tra.Int() < u.MinTra {
		return false
	}
	return true
}

func (a *Atlas) folderUnlocked(folder string, city *citycore.City) bool {
	if a == nil || city == nil {
		return true
	}
	if a.UnlockByFolder == nil {
		return true
	}
	u, ok := a.UnlockByFolder[folder]
	if !ok {
		return true
	}
	return u.satisfied(city)
}

func filterBasesByUnlock(a *Atlas, bases []string, city *citycore.City) []string {
	if a == nil || city == nil || len(a.UnlockByFolder) == 0 {
		return bases
	}
	out := make([]string, 0, len(bases))
	for _, base := range bases {
		if a.folderUnlocked(spriteFolderBase(base), city) {
			out = append(out, base)
		}
	}
	return out
}

func filterFoldersByUnlock(a *Atlas, folders []string, city *citycore.City) []string {
	if a == nil || city == nil || len(a.UnlockByFolder) == 0 {
		return folders
	}
	out := make([]string, 0, len(folders))
	for _, folder := range folders {
		if a.folderUnlocked(folder, city) {
			out = append(out, folder)
		}
	}
	return out
}
