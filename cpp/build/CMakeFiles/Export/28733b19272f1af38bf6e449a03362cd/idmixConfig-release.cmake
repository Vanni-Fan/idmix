#----------------------------------------------------------------
# Generated CMake target import file for configuration "Release".
#----------------------------------------------------------------

# Commands may need to know the format version.
set(CMAKE_IMPORT_FILE_VERSION 1)

# Import target "idmix::idmix" for configuration "Release"
set_property(TARGET idmix::idmix APPEND PROPERTY IMPORTED_CONFIGURATIONS RELEASE)
set_target_properties(idmix::idmix PROPERTIES
  IMPORTED_LINK_INTERFACE_LANGUAGES_RELEASE "CXX"
  IMPORTED_LOCATION_RELEASE "${_IMPORT_PREFIX}/lib/idmix.lib"
  )

list(APPEND _cmake_import_check_targets idmix::idmix )
list(APPEND _cmake_import_check_files_for_idmix::idmix "${_IMPORT_PREFIX}/lib/idmix.lib" )

# Commands beyond this point should not need to know the version.
set(CMAKE_IMPORT_FILE_VERSION)
