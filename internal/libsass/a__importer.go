// Copyright © 2018 Bjørn Erik Pedersen <bjorn.erik.pedersen@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package libsass

// #include <stdint.h>
// #include <stdlib.h>
// #include <string.h>
// #include "sass/context.h"
//
// extern struct Sass_Import** ResolveImportBridge(const char* url, const char* prev, uintptr_t idx);
//
// Sass_Import_List SassImporterHandler(const char* cur_path, Sass_Importer_Entry cb, struct Sass_Compiler* comp)
// {
//   void* cookie = sass_importer_get_cookie(cb);
//   struct Sass_Import* previous = sass_compiler_get_last_import(comp);
//   const char* prev_path = sass_import_get_imp_path(previous);
//   uintptr_t idx = (uintptr_t)cookie;
//   Sass_Import_List list = ResolveImportBridge(cur_path, prev_path, idx);
//   return list;
// }
//
//
// #ifndef UINTMAX_MAX
// #  ifdef __UINTMAX_MAX__
// #    define UINTMAX_MAX __UINTMAX_MAX__
// #  endif
// #endif
//
// //size_t max_size = UINTMAX_MAX;
import "C"
import (
	"sync"
	"unsafe"
)

// ImportResolver can be used as a custom import resolver. Return an empty body to
// signal loading the import body from the URL.
type ImportResolver func(url string, prev string) (newURL string, body string, resolved bool)

// AddImportResolver adds a function to resolve imports in LibSASS.
// Make sure to run call DeleteImportResolver whendone.
func AddImportResolver(opts SassOptions, resolver ImportResolver) int {

	i := imports.Set(resolver)
	ptr := unsafe.Pointer(uintptr(i))

	imper := C.sass_make_importer(
		C.Sass_Importer_Fn(C.SassImporterHandler),
		C.double(0),
		ptr,
	)
	impers := C.sass_make_importer_list(1)
	C.sass_importer_set_list_entry(impers, 0, imper)

	C.sass_option_set_c_importers(
		(*C.struct_Sass_Options)(unsafe.Pointer(opts)),
		impers,
	)

	return i
}

func DeleteImportResolver(i int) error {
	imports.Delete(i)
	return nil
}

var imports = &idMap{
	m: make(map[int]interface{}),
}

type idMap struct {
	sync.RWMutex
	m map[int]interface{}
	i int
}

func (m *idMap) Get(i int) interface{} {
	m.RLock()
	defer m.RUnlock()
	return m.m[i]
}

func (m *idMap) Set(v interface{}) int {
	m.Lock()
	defer m.Unlock()
	m.i++
	m.m[m.i] = v
	return m.i
}

func (m *idMap) Delete(i int) {
	m.Lock()
	defer m.Unlock()
	delete(m.m, i)
}