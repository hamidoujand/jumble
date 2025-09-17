package page

import (
	"fmt"
	"strconv"
)

type Page struct {
	Number int
	Rows   int
}

func Parse(pageNumber string, rowsPerPage string) (Page, error) {
	//default page number
	number := 1
	//default rows per page
	rows := 10

	if pageNumber != "" {
		//convert into num
		var err error
		number, err = strconv.Atoi(pageNumber)
		if err != nil {
			return Page{}, fmt.Errorf("converting page number: %w", err)
		}
	}

	if rowsPerPage != "" {
		var err error
		rows, err = strconv.Atoi(rowsPerPage)
		if err != nil {
			return Page{}, fmt.Errorf("converting rows per page: %w", err)
		}
	}

	if number <= 0 {
		return Page{}, fmt.Errorf("%d, value too small, must be greater than 0", number)
	}

	if rows <= 0 {
		return Page{}, fmt.Errorf("%d, value too small, must be greater than 0", rows)
	}

	if number > 100 {
		return Page{}, fmt.Errorf("%d, value too big, must be less than 100", number)
	}

	return Page{Number: number, Rows: rows}, nil
}
