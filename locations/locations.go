package locations

import (
	"fmt"
	"sort"
	"strings"
)

// Location представляет информацию о локации VPN
type Location struct {
	ISO        string
	Country    string
	City       string
	Ping       int
	Bookmarked bool
}

// SortColumn определяет столбец для сортировки
type SortColumn int

const (
	SortByISO SortColumn = iota
	SortByCountry
	SortByCity
	SortByPing
)

// SortLocations сортирует локации по указанному столбцу
func SortLocations(locs []Location, column SortColumn, ascending bool) []Location {
	result := make([]Location, len(locs))
	copy(result, locs)

	sort.Slice(result, func(i, j int) bool {
		var less bool
		switch column {
		case SortByISO:
			less = strings.ToLower(result[i].ISO) < strings.ToLower(result[j].ISO)
		case SortByCountry:
			less = strings.ToLower(result[i].Country) < strings.ToLower(result[j].Country)
		case SortByCity:
			less = strings.ToLower(result[i].City) < strings.ToLower(result[j].City)
		case SortByPing:
			less = result[i].Ping < result[j].Ping
		default:
			less = result[i].Ping < result[j].Ping
		}
		if ascending {
			return less
		}
		return !less
	})

	return result
}

// ApplyBookmarkFlags returns a copy of locs with Bookmarked set from the lookup function.
func ApplyBookmarkFlags(locs []Location, bookmarked func(Location) bool) []Location {
	result := make([]Location, len(locs))
	copy(result, locs)
	for i := range result {
		result[i].Bookmarked = bookmarked(result[i])
	}
	return result
}

// SortLocationsWithBookmarks sorts locations and optionally moves bookmarked entries first.
// When bookmarksFirst is false, the result matches SortLocations.
func SortLocationsWithBookmarks(locs []Location, column SortColumn, ascending bool, bookmarksFirst bool) []Location {
	result := SortLocations(locs, column, ascending)
	if !bookmarksFirst {
		return result
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Bookmarked == result[j].Bookmarked {
			return false
		}
		return result[i].Bookmarked
	})

	return result
}

// ParseLocations парсит вывод команды list-locations
func ParseLocations(output string) []Location {
	lines := strings.Split(output, "\n")
	var locations []Location

	// Находим позиции колонок по заголовку
	var isoEnd, countryEnd, cityEnd int
	for _, line := range lines {
		cleanLine := strings.ReplaceAll(line, "\x1b[1m", "")
		cleanLine = strings.ReplaceAll(cleanLine, "\x1b[0m", "")
		if strings.Contains(cleanLine, "ISO") && strings.Contains(cleanLine, "COUNTRY") {
			isoEnd = strings.Index(cleanLine, "COUNTRY")
			countryEnd = strings.Index(cleanLine, "CITY")
			cityEnd = strings.Index(cleanLine, "PING")
			break
		}
	}

	if isoEnd == 0 || countryEnd == 0 || cityEnd == 0 {
		fmt.Println("Could not determine column positions from header")
		return locations
	}

	for _, line := range lines {
		// Убираем ANSI escape codes
		cleanLine := strings.ReplaceAll(line, "\x1b[1m", "")
		cleanLine = strings.ReplaceAll(cleanLine, "\x1b[0m", "")

		// Пропускаем пустые строки и заголовок
		if strings.TrimSpace(cleanLine) == "" || strings.Contains(cleanLine, "ISO") {
			continue
		}

		// Извлекаем данные, используя найденные позиции колонок
		var iso, country, city, pingStr string

		// Безопасное извлечение подстрок
		if len(cleanLine) > isoEnd {
			iso = strings.TrimSpace(cleanLine[:isoEnd])
		}
		if len(cleanLine) > countryEnd {
			country = strings.TrimSpace(cleanLine[isoEnd:countryEnd])
		}
		if len(cleanLine) > cityEnd {
			city = strings.TrimSpace(cleanLine[countryEnd:cityEnd])
			pingStr = strings.TrimSpace(cleanLine[cityEnd:])
		} else if len(cleanLine) > countryEnd {
			// Если строка короче, но хватает до countryEnd
			city = strings.TrimSpace(cleanLine[countryEnd:])
			pingStr = ""
		}

		// Пропускаем строки с невалидными данными
		if iso == "" || country == "" || city == "" {
			continue
		}

		// Конвертируем пинг в число
		var ping int
		if pingStr != "" {
			if _, err := fmt.Sscanf(pingStr, "%d", &ping); err != nil {
				// Если не удалось сконвертировать, устанавливаем максимальное значение
				ping = 9999
			}
		} else {
			ping = 9999
		}

		locations = append(locations, Location{
			ISO:     iso,
			Country: country,
			City:    city,
			Ping:    ping,
		})
	}

	// Сортируем локации по пингу (от меньшего к большему)
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].Ping < locations[j].Ping
	})

	return locations
}

// FilterLocations фильтрует локации по ISO-коду, имени города или страны
func FilterLocations(locations []Location, query string) []Location {
	query = strings.TrimSpace(query)
	if query == "" {
		return locations
	}

	var filtered []Location
	query = strings.ToLower(query)

	for _, loc := range locations {
		if strings.Contains(strings.ToLower(loc.ISO), query) ||
			strings.Contains(strings.ToLower(loc.City), query) ||
			strings.Contains(strings.ToLower(loc.Country), query) {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}

// FindByCity returns the first location whose city matches case-insensitively.
func FindByCity(locs []Location, city string) *Location {
	target := strings.TrimSpace(city)
	if target == "" {
		return nil
	}
	for i := range locs {
		if strings.EqualFold(locs[i].City, target) {
			return &locs[i]
		}
	}
	return nil
}

// FindFastestLocation находит локацию с минимальным пингом
func FindFastestLocation(locations []Location) *Location {
	if len(locations) == 0 {
		return nil
	}

	fastest := &locations[0]
	for i := 1; i < len(locations); i++ {
		if locations[i].Ping < fastest.Ping {
			fastest = &locations[i]
		}
	}
	return fastest
}
