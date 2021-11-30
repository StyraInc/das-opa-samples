package rules

# Load into a custom system in the rules package

import data.dataset

default main = false

main {
  path_parts = ["something", "allow"]
}

path_parts = p {
  p := split(trim(input.path, "/"), "/")
}
