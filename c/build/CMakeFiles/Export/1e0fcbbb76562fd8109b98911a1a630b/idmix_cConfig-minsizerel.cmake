#----------------------------------------------------------------
# Generated CMake target import file for configuration "MinSizeRel".
#----------------------------------------------------------------

# Commands may need to know the format version.
set(CMAKE_IMPORT_FILE_VERSION 1)

# Import target "idmix::idmix_c" for configuration "MinSizeRel"
set_property(TARGET idmix::idmix_c APPEND PROPERTY IMPORTED_CONFIGURATIONS MINSIZEREL)
set_target_properties(idmix::idmix_c PROPERTIES
  IMPORTED_LINK_INTERFACE_LANGUAGES_MINSIZEREL "C"
  IMPORTED_LOCATION_MINSIZEREL "${_IMPORT_PREFIX}/lib/idmix_c.lib"
  )

list(APPEND _cmake_import_check_targets idmix::idmix_c )
list(APPEND _cmake_import_check_files_for_idmix::idmix_c "${_IMPORT_PREFIX}/lib/idmix_c.lib" )

# Commands beyond this point should not need to know the version.
set(CMAKE_IMPORT_FILE_VERSION)
