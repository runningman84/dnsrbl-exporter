#!/usr/bin/env python
# -*- coding: utf-8 -*-

from setuptools import setup, find_packages

REQUIRES = [
    'prometheus_client',
    'urllib3',
    'dnspython',
]

setup(
    name='dnsrbl-exporter',
    version='1.0',
    description="DNS Realtime blacklist exporter",
    long_description="dnsrbl-exporter is a dns realtime blacklist checker with a prometheus endpoint",
    author='Philipp Hellmich',
    author_email='phil@hellmi.de',
    url='https://github.com/runningman84/dnsrbl-exporter.git',
    install_requires=REQUIRES,
    license='Apache License 2.0',
    zip_safe=False,
    keywords=['dnsrbl-exporter'],
    classifiers=[
        'Development Status :: 5 - Production/Stable',
        'Intended Audience :: Developers',
        'License :: OSI Approved :: Apache Software License',
        'Natural Language :: English',
        'Programming Language :: Python :: 3.4',
        'Programming Language :: Python :: 3.5',
        'Programming Language :: Python :: 3.6'
    ],
)