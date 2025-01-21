from setuptools import setup, find_packages

setup(
    name='intertrans',
    version='0.1',
    packages=find_packages(),
    install_requires=[],
    author='Marcos Macedo',
    author_email='marcos.macedo@queensu.ca',
    description='InterTrans Client',
    long_description='',
    long_description_content_type='text/markdown',
    url='https://github.com/RISElabQueens/intertrans',
    classifiers=[
        'Development Status :: 3 - Alpha',
        'License :: OSI Approved :: MIT License',
    ],
    python_requires='>=3.7',
)