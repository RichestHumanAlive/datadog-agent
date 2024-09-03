"""
This module provides functions for managing Datadog integrations and Python dependencies after installation

Usage:
- The script should be run with a single argument specifying the installation directory.
- Example: `python postinst.py /path/to/install/dir`
"""

import os
import sys
import packages


def create_python_installed_packages_file(directory):
    postinst_python_installed_packages_file = packages.postinst_python_installed_packages_file(directory)
    print(f"Creating file: {postinst_python_installed_packages_file}")
    packages.create_python_installed_packages_file(postinst_python_installed_packages_file)

def install_diff_packages_file(directory):
    diff_python_installed_packages_file = packages.diff_python_installed_packages_file(directory)
    if os.path.exists(diff_python_installed_packages_file):
        print(f"Installing python packages from: {diff_python_installed_packages_file}")
        packages.install_diff_packages_file(f"{directory}/embedded/bin/pip", diff_python_installed_packages_file)
        packages.cleanup_files(diff_python_installed_packages_file)
    else:
        print(f"File {diff_python_installed_packages_file} does not exist.")

if __name__ == '__main__':
    if len(sys.argv) != 2:
        print("Usage: postinst.py <INSTALL_DIR>")
        sys.exit(1)
    install_directory = sys.argv[1]
    if os.path.exists(install_directory):
        create_python_installed_packages_file(install_directory)
        ENV_VAR = 'INSTALL_PYTHON_THIRD_PARTY_DEPS'
        if ENV_VAR in os.environ:
            env_var_value = os.environ[ENV_VAR].lower() in ['true', '1', 't', 'yes', 'y']
            if env_var_value:
                install_diff_packages_file(install_directory)
            else:
                print(f"{ENV_VAR} is set to: {env_var_value}")
        else:
            print(f"{ENV_VAR} is not set.")
    else:
        print(f"Directory {install_directory} does not exist.")
        sys.exit(1)
