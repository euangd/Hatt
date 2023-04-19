package specificScrapers

import (
	"fmt"
	"hatt/assets"
	"hatt/helpers"
	"hatt/login"
	"hatt/variables"
	"strings"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// using go-rod instead of colly because the website checks on many things (headers/cookies etc) and I can't find the good combination of these to send requests without a real browser without being flagged
func (t T) Simpcity() []variables.Item {

	var results []variables.Item

	config := assets.DeserializeWebsiteConf("simpcity.json")
	loginSuccessfull := login.LoginBrowser("simpcity")
	if !loginSuccessfull {
		message := variables.Item{
			Name: "error",
			Metadata: map[string]string{
				"name": "login_required",
			},
		}
		results = append(results, message)
		return results
	}

	h := &helpers.Helper{}
	tokens := h.DeserializeCredentials("simpcity").Tokens

	// send search request, a url is generated by the website with a unique id
	l := helpers.InstanciateBrowser()
	cookies := []*proto.NetworkCookieParam{}
	for tokenName, token := range tokens {
		cookies = append(cookies, &proto.NetworkCookieParam{
			Name:   tokenName,
			Value:  token["value"],
			Domain: config.SpecificInfo["domain"],
		})
	}

	browser := rod.New().ControlURL(l).MustConnect()
	browser.SetCookies(cookies)

	page := browser.NoDefaultDevice().MustPage(config.Search.Url)
	page.MustWaitLoad()

	searchInput := page.MustElement(".inputList li:nth-of-type(1) input")
	searchInput.MustClick().MustSelectAllText().MustInput(variables.CURRENT_INPUT)

	var wg sync.WaitGroup
	wg.Add(1)
	go page.EachEvent(func(e *proto.PageLoadEventFired) {
		// page loaded
		wg.Done()
	})()
	// setting the page's viewPort bigger, otherwise the buttons are not accessible/not working somehow
	viewPort := proto.EmulationSetDeviceMetricsOverride{
		Width:  1920,
		Height: 1080,
	}
	page.SetViewport(&viewPort)
	searchButton := page.MustElement(".formSubmitRow-main button")
	clickError := searchButton.Click(proto.InputMouseButtonLeft, 1)
	if clickError != nil {
		fmt.Println("error when clicking on search : ", clickError)
	}
	wg.Wait()

	itemKeys := config.Search.ItemKeys
	for _, result := range page.MustElements(itemKeys.Root) {
		item := variables.Item{
			Name:     result.MustElement(itemKeys.Name).MustText(),
			Link:     config.Login.HomeUrl + *result.MustElement(itemKeys.Link).MustAttribute("href"),
			Metadata: map[string]string{},
		}
		for _, li := range result.MustElements(".contentRow-minor ul li") {
			if strings.Contains(li.MustText(), "Replies:") {
				item.Metadata["replies"] = li.MustText()
			} else if strings.Contains(li.MustText(), " at ") {
				item.Metadata["postedAt"] = li.MustText()
			}
		}

		results = append(results, item)
	}

	// now that the search url has been retreived, no cookies/headers etc. are needed to view the search results, so using colly instead

	// for index, item := range results {
	// 	wg.Add(1)
	// 	go func(item variables.Item, index int) {
	// 		pageCollector := colly.NewCollector()
	// 		imgUrls := []string{}

	// 		// todo : if the item's link contains "post-xxxx", then only check for images in the specific post and not in the whole thread
	// 		pageCollector.OnHTML("img.bbImage", func(h *colly.HTMLElement) {
	// 			// store urls in list, then loop over list and take the first image that works
	// 			imgUrls = append(imgUrls, h.Attr("src"))
	// 		})

	// 		pageCollector.Visit(item.Link)

	// 		for _, url := range imgUrls {
	// 			imgBase64 := helpers.GetImageBase64(url, nil)
	// 			results[index].Thumbnail = imgBase64
	// 			wg.Done()
	// 			return
	// 		}
	// 		// if no image was retreived
	// 		wg.Done()
	// 		return

	// 	}(item, index)

	// }
	// wg.Wait()
	browser.Close()

	return results
}
