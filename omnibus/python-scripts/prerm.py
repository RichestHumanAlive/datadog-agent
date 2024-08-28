"""
This module handles the cleanup of Datadog integrations and Python dependencies during package removal.

Functions:
- run_command(command): Executes a shell command and returns its output.
- create_new_integrations_file(directory): Creates a temporary file listing currently installed Datadog integrations.
- create_new_dependencies_file(directory): Creates a temporary file listing currently installed Python dependencies.
- compare_and_update_files(old_file, new_file, output_file): Compares old and new files and writes differences to an output file.
- cleanup_files(*files): Removes the specified files.

Usage:
- The script should be run with a single argument specifying the installation directory.
- Example: `python script_prerm.py /path/to/install/dir`
"""

import os
import subprocess
import shutil
import sys

import pkg_resources
from packaging import version
import importlib.metadata
import pwd
import grp
import packages

def run_command(command):
    """
    Execute a shell command and return its output.

    Args:
        command (str): The shell command to execute.

    Returns:
        str: The standard output of the command.

    Raises:
        subprocess.CalledProcessError: If the command exits with a non-zero status.
    """
    result = subprocess.run(command, shell=True, text=True,
                            capture_output=True, check=True)
    return result.stdout.strip()

def create_new_integrations_file(directory):
    """
    Create a temporary file listing the currently installed Datadog integrations.

    This function runs the `datadog-agent integration freeze` command, sorts the output,
    and writes it to a file named '.new_datadog_requirements.txt' in the specified directory.

    Args:
        directory (str): The directory where the temporary integrations file will be created.
    """
    print("Creating installed integrations file")
    new_datadog_file = os.path.join(directory, '.new_datadog_requirements.txt')
    
    output = run_command('datadog-agent integration freeze')
    sorted_output = '\n'.join(sorted(output.splitlines()))
    
    with open(new_datadog_file, 'w', encoding='utf-8') as f:
        f.write(sorted_output)
    
    shutil.chown(new_datadog_file, user='dd-agent', group='dd-agent')
    return new_datadog_file

def create_new_dependencies_file(directory):
    """
    Create a temporary file listing the currently installed Python dependencies, excluding Datadog packages.

    This function runs the `pip freeze` command, filters out Datadog packages, sorts the output,
    and writes it to a file named '.new_python_requirements.txt' in the specified directory.

    Args:
        directory (str): The directory where the temporary dependencies file will be created.
    """
    print("Creating installed dependencies file")
    new_python_file = os.path.join(directory, '.new_python_requirements.txt')
    
    output = run_command(f'{directory}/embedded/bin/pip list --format=freeze | grep -v "^datadog-"')
    sorted_output = '\n'.join(sorted(output.splitlines()))

    with open(new_python_file, 'w', encoding='utf-8') as f:
        f.write(sorted_output)
    
    shutil.chown(new_python_file, user='dd-agent', group='dd-agent')
    return new_python_file

def compare_and_update_files(old_file, new_file, output_file):
    """
    Compare the old and new requirements files, writing any differences to an output file.

    This function uses the `comm -13` command to find lines present in the new file but not in the old file.
    The differences are written to the output file.

    Args:
        old_file (str): The path to the old requirements file.
        new_file (str): The path to the new temporary requirements file.
        output_file (str): The path to the output file that will contain the differences.
    """
    if os.path.exists(old_file):
        print(f"Comparing {old_file} with {new_file}")
        run_command(f'comm -13 {old_file} {new_file} > {output_file}')

def cleanup_files(*files):
    """
    Remove the specified files.

    Args:
        files (str): Paths to the files that should be removed.
    """
    for file in files:
        if os.path.exists(file):
            print(f"Removing file: {file}")
            os.remove(file)

def load_requirements(filename):
    """
    Load requirements from a file.
    """
    with open(filename, 'r', encoding='utf-8') as f:
        return list(pkg_resources.parse_requirements(f))

def get_requirements_dict(requirements):
    """
    Create a dictionary from requirements with package names as keys and versions as values.
    """
    return {req.name: req for req in requirements}

def extract_version(specifier):
    """
    Extract version from the specifier string.
    """
    try:
        # Get the first version specifier from the specifier string
        return str(next(iter(pkg_resources.Requirement.parse(f'{specifier}').specifier)))
    except Exception:
        return None
    
if __name__ == '__main__':
    if len(sys.argv) != 2:
        print("Usage: prerm.py <INSTALL_DIR>")
        sys.exit(1)

    install_directory = sys.argv[1]
    
    if os.path.exists(install_directory):
        postinst_python_installed_packages_file = os.path.join(install_directory, '.postinst_python_installed_packages.txt')
        if os.path.exists(postinst_python_installed_packages_file):
            prerm_python_installed_packages_file = os.path.join(install_directory, '.prerm_python_installed_packages.txt')
            packages.create_python_installed_packages_file(prerm_python_installed_packages_file)

            postinst_python_installed_packages = packages.load_requirements(postinst_python_installed_packages_file)
            prerm_python_installed_packages = packages.load_requirements(prerm_python_installed_packages_file)

            # Find packages that are new or have been upgraded
            updated_requirements = []
            for package_name, new_req in prerm_python_installed_packages.items():
                old_req = postinst_python_installed_packages.get(package_name)
                if old_req:
                    # Extract and compare versions
                    old_version_str = extract_version(str(old_req.specifier))
                    new_version_str = extract_version(str(new_req.specifier))
                    if old_version_str and new_version_str:
                        if version.parse(new_version_str) > version.parse(old_version_str):
                            updated_requirements.append(new_req)
                else:
                    # Package is new in the new file; include it
                    updated_requirements.append(new_req)
            print(f"updated_requirements: {updated_requirements}")
        else:
            print(f"File {postinst_python_installed_packages_file} does not exist.")
            sys.exit(1)
    else:
        print(f"Directory {install_directory} does not exist.")
        sys.exit(1)