package test

import "testing"

func TestGetUniquePartitions(t *testing.T) {
	cases := []struct {
		name               string
		isConsulEnterprise bool
		expectedPartitions int
		err                bool
	}{
		{name: "CE", isConsulEnterprise: false, expectedPartitions: 1, err: false},
		{name: "ENT", isConsulEnterprise: true, expectedPartitions: 2, err: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tenancyHelper := &TenancyHelper{isConsulEnterprise: c.isConsulEnterprise}
			partitions := tenancyHelper.GetUniquePartitions()
			if len(partitions) != c.expectedPartitions && !c.err {
				t.Fatalf("expected %d partitions, got %d", c.expectedPartitions, len(partitions))
			}
		})
	}
}

func TestAppendTenancyInfo(t *testing.T) {
	cases := []struct {
		name      string
		namespace string
		partition string
		expected  string
	}{
		{name: "empty", namespace: "", partition: "", expected: "empty__Namespace__Partition"},
		{name: "namespace", namespace: "foo", partition: "", expected: "namespace_foo_Namespace__Partition"},
		{name: "partition", namespace: "", partition: "bar", expected: "partition__Namespace_bar_Partition"},
		{name: "both", namespace: "foo", partition: "bar", expected: "both_foo_Namespace_bar_Partition"},
	}

	tenancyHelper := &TenancyHelper{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tenancy := &Tenancy{Namespace: c.namespace, Partition: c.partition}
			result := tenancyHelper.AppendTenancyInfo(c.name, tenancy)
			if result != c.expected {
				t.Fatalf("expected %q, got %q", c.expected, result)
			}
		})
	}
}

func TestTenancy(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected *Tenancy
	}{
		{name: "empty", input: "", expected: &Tenancy{}},
		{name: "partition", input: "foo", expected: &Tenancy{Partition: "foo"}},
		{name: "both", input: "foo.bar", expected: &Tenancy{Partition: "foo", Namespace: "bar"}},
		{name: "bad", input: "foo.bar.baz", expected: &Tenancy{Partition: "BAD", Namespace: "BAD"}},
	}

	tenancyHelper := &TenancyHelper{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := tenancyHelper.Tenancy(c.input)
			if *result != *c.expected {
				t.Fatalf("expected %v, got %v", c.expected, result)
			}
		})
	}
}

func TestTenancies(t *testing.T) {
	cases := []struct {
		name               string
		isConsulEnterprise bool
		expectedTenancies  int
	}{
		{name: "CE", isConsulEnterprise: false, expectedTenancies: 1},
		{name: "ENT", isConsulEnterprise: true, expectedTenancies: 4},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tenancyHelper := &TenancyHelper{isConsulEnterprise: c.isConsulEnterprise}
			tenancies := tenancyHelper.TestTenancies()
			if len(tenancies) != c.expectedTenancies {
				t.Fatalf("expected %d tenancies, got %d", c.expectedTenancies, len(tenancies))
			}
		})
	}
}

func TestGenerateTenancyTests(t *testing.T) {
	type fakeTest struct {
		name string
	}
	cases := []struct {
		name               string
		isConsulEnterprise bool
		expectedTests      int
		input              func(tenancy *Tenancy) []interface{}
	}{
		{name: "CE", isConsulEnterprise: false, expectedTests: 2, input: func(tenancy *Tenancy) []interface{} {
			return []interface{}{
				fakeTest{name: "CE1"},
				fakeTest{name: "CE2"},
			}
		}},
		{name: "ENT", isConsulEnterprise: true, expectedTests: 8, input: func(tenancy *Tenancy) []interface{} {
			return []interface{}{
				fakeTest{name: "ENT1"},
				fakeTest{name: "ENT2"},
			}
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tenancyHelper := &TenancyHelper{isConsulEnterprise: c.isConsulEnterprise}
			tests := tenancyHelper.GenerateTenancyTests(c.input)
			if len(tests) != c.expectedTests {
				t.Fatalf("expected %d tests, got %d", c.expectedTests, len(tests))
			}
		})
	}
}

func TestGenerateDefaultTenancyTests(t *testing.T) {
	type fakeTest struct {
		name string
	}
	cases := []struct {
		name               string
		isConsulEnterprise bool
		expectedTests      int
		input              func(tenancy *Tenancy) []interface{}
	}{
		{name: "CE", isConsulEnterprise: false, expectedTests: 2, input: func(tenancy *Tenancy) []interface{} {
			return []interface{}{
				fakeTest{name: "CE1"},
				fakeTest{name: "CE2"},
			}
		}},
		{name: "ENT", isConsulEnterprise: true, expectedTests: 2, input: func(tenancy *Tenancy) []interface{} {
			return []interface{}{
				fakeTest{name: "ENT1"},
				fakeTest{name: "ENT2"},
			}
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tenancyHelper := &TenancyHelper{isConsulEnterprise: c.isConsulEnterprise}
			tests := tenancyHelper.GenerateDefaultTenancyTests(c.input)
			if len(tests) != c.expectedTests {
				t.Fatalf("expected %d tests, got %d", c.expectedTests, len(tests))
			}
		})
	}
}

func TestGenerateNonDefaultTenancyTests(t *testing.T) {
	type fakeTest struct {
		name string
	}
	cases := []struct {
		name               string
		isConsulEnterprise bool
		expectedTests      int
		input              func(tenancy *Tenancy) []interface{}
	}{
		{name: "CE", isConsulEnterprise: false, expectedTests: 0, input: func(tenancy *Tenancy) []interface{} {
			return []interface{}{
				fakeTest{name: "CE1"},
				fakeTest{name: "CE2"},
			}
		}},
		{name: "ENT", isConsulEnterprise: true, expectedTests: 6, input: func(tenancy *Tenancy) []interface{} {
			return []interface{}{
				fakeTest{name: "ENT1"},
				fakeTest{name: "ENT2"},
			}
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tenancyHelper := &TenancyHelper{isConsulEnterprise: c.isConsulEnterprise}
			tests := tenancyHelper.GenerateNonDefaultTenancyTests(c.input)
			if len(tests) != c.expectedTests {
				t.Fatalf("expected %d tests, got %d", c.expectedTests, len(tests))
			}
		})
	}
}
