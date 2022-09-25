module vendoring.example.com/a

go 1.19

require example.com/replaced v0.1.0

replace example.com/replaced v0.1.0 => ../_replaced_module
