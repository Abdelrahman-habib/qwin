package services

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"qwin/internal/types"

	"github.com/gocolly/colly/v2"
)

// LinkedInScraper handles scraping LinkedIn Learning courses
type LinkedInScraper struct {
	collector *colly.Collector
}

// NewLinkedInScraper creates a new LinkedIn scraper instance
func NewLinkedInScraper() *LinkedInScraper {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	// Set reasonable limits
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*linkedin.com*",
		Parallelism: 2,
		Delay:       1 * time.Second,
	})

	return &LinkedInScraper{
		collector: c,
	}
}

// ScrapeCourse scrapes a LinkedIn Learning course from the given URL
func (ls *LinkedInScraper) ScrapeCourse(courseURL string) (*types.ScrapingResult, error) {
	// Validate URL
	if !ls.isValidLinkedInLearningURL(courseURL) {
		return &types.ScrapingResult{
			Success: false,
			Error:   "Invalid LinkedIn Learning URL",
		}, nil
	}

	course := types.Course{
		URL:       courseURL,
		CreatedAt: time.Now(),
	}

	var scrapingError error

	// Extract course ID from URL
	course.ID = ls.extractCourseID(courseURL)

	// Set up scraping handlers
	ls.collector.OnHTML("h1.course-header__title", func(e *colly.HTMLElement) {
		course.Title = strings.TrimSpace(e.Text)
	})

	ls.collector.OnHTML(".course-header__description", func(e *colly.HTMLElement) {
		course.Description = strings.TrimSpace(e.Text)
	})

	ls.collector.OnHTML(".course-header__instructor-name", func(e *colly.HTMLElement) {
		course.Instructor = strings.TrimSpace(e.Text)
	})

	ls.collector.OnHTML(".course-header__duration", func(e *colly.HTMLElement) {
		course.Duration = strings.TrimSpace(e.Text)
	})

	ls.collector.OnHTML(".course-header__level", func(e *colly.HTMLElement) {
		course.Level = strings.TrimSpace(e.Text)
	})

	// Scrape chapters and sections
	chapterOrder := 0
	ls.collector.OnHTML(".course-toc__chapter", func(e *colly.HTMLElement) {
		chapter := types.Chapter{
			Order: chapterOrder,
		}
		chapterOrder++

		// Chapter title
		e.ForEach(".course-toc__chapter-title", func(_ int, el *colly.HTMLElement) {
			chapter.Title = strings.TrimSpace(el.Text)
			chapter.ID = ls.generateChapterID(course.ID, chapter.Order)
		})

		// Chapter duration
		e.ForEach(".course-toc__chapter-duration", func(_ int, el *colly.HTMLElement) {
			chapter.Duration = strings.TrimSpace(el.Text)
		})

		// Sections within this chapter
		sectionOrder := 0
		e.ForEach(".course-toc__item", func(_ int, sectionEl *colly.HTMLElement) {
			section := types.Section{
				Order: sectionOrder,
			}
			sectionOrder++

			// Section title
			sectionEl.ForEach(".course-toc__item-title", func(_ int, titleEl *colly.HTMLElement) {
				section.Title = strings.TrimSpace(titleEl.Text)
				section.ID = ls.generateSectionID(chapter.ID, section.Order)
			})

			// Section duration
			sectionEl.ForEach(".course-toc__item-duration", func(_ int, durationEl *colly.HTMLElement) {
				section.Duration = strings.TrimSpace(durationEl.Text)
			})

			// Section URL
			sectionEl.ForEach("a", func(_ int, linkEl *colly.HTMLElement) {
				href := linkEl.Attr("href")
				if href != "" {
					section.URL = ls.buildAbsoluteURL(courseURL, href)
				}
			})

			if section.Title != "" {
				chapter.Sections = append(chapter.Sections, section)
			}
		})

		if chapter.Title != "" {
			course.Chapters = append(course.Chapters, chapter)
		}
	})

	ls.collector.OnError(func(r *colly.Response, err error) {
		scrapingError = fmt.Errorf("scraping error: %v", err)
	})

	// Visit the course page
	err := ls.collector.Visit(courseURL)
	if err != nil {
		return &types.ScrapingResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to visit URL: %v", err),
		}, err
	}

	if scrapingError != nil {
		return &types.ScrapingResult{
			Success: false,
			Error:   scrapingError.Error(),
		}, scrapingError
	}

	// Validate that we got some data
	if course.Title == "" {
		return &types.ScrapingResult{
			Success: false,
			Error:   "Could not extract course data. The page structure might have changed or the course might not be accessible.",
		}, nil
	}

	return &types.ScrapingResult{
		Success: true,
		Course:  course,
	}, nil
}

// isValidLinkedInLearningURL validates if the URL is a LinkedIn Learning course URL
func (ls *LinkedInScraper) isValidLinkedInLearningURL(courseURL string) bool {
	u, err := url.Parse(courseURL)
	if err != nil {
		return false
	}

	return strings.Contains(u.Host, "linkedin.com") && strings.Contains(u.Path, "/learning/")
}

// extractCourseID extracts course ID from LinkedIn Learning URL
func (ls *LinkedInScraper) extractCourseID(courseURL string) string {
	// Extract course ID from URL pattern like /learning/course-name-12345/
	re := regexp.MustCompile(`/learning/([^/]+)/?`)
	matches := re.FindStringSubmatch(courseURL)
	if len(matches) > 1 {
		return matches[1]
	}
	return fmt.Sprintf("course_%d", time.Now().Unix())
}

// generateChapterID generates a unique chapter ID
func (ls *LinkedInScraper) generateChapterID(courseID string, order int) string {
	return fmt.Sprintf("%s_chapter_%d", courseID, order)
}

// generateSectionID generates a unique section ID
func (ls *LinkedInScraper) generateSectionID(chapterID string, order int) string {
	return fmt.Sprintf("%s_section_%d", chapterID, order)
}

// buildAbsoluteURL builds absolute URL from relative URL
func (ls *LinkedInScraper) buildAbsoluteURL(baseURL, relativeURL string) string {
	if strings.HasPrefix(relativeURL, "http") {
		return relativeURL
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return relativeURL
	}

	rel, err := url.Parse(relativeURL)
	if err != nil {
		return relativeURL
	}

	return base.ResolveReference(rel).String()
}
