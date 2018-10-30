package client

import "testing"

func TestGetByName(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	r, err := c.GetApiResourceByName("services", "")
	check(t ,err)
	assert(t, r.Kind, "Service")


	r, err = c.GetApiResourceByName("service", "")
	print(err.Error())
	assert(t, IsResourceTypeNotFindError(err), true)
}
