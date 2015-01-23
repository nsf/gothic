package gothic

type handle struct {
	ID    int
	Value interface{}
}

type handles []handle

func (h *handles) init_maybe() {
	if *h == nil {
		*h = make(handles, 1)
	}
	if len(*h) == 0 {
		*h = append(*h, handle{0, nil})
	}
}

func (h *handles) get_handle_for_value(value interface{}) int {
	h.init_maybe()
	hh := *h
	free_id := hh[0].ID
	if free_id != 0 {
		hh[0].ID = hh[free_id].ID
		hh[free_id].Value = value
		return free_id
	}

	*h = append(hh, handle{-1, value})
	return len(*h) - 1
}

func (h *handles) free_handle(id int) {
	hh := *h
	hh[id].Value = nil
	hh[id].ID = hh[0].ID
	hh[0].ID = id
}
