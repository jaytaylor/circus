#!/usr/bin/env python
# -*- coding: utf-8 -*-

import json
import logging
import os
import os.path
import subprocess
import sys

def main(args):
	if len(args) < 2:
		sys.stderr.write('ERROR: Missing required parameter: [input-file.json]\n')
		sys.exit(1)

	if args[1] == '-':
		fh = sys.stdin
	else:
		fh = open(args[1], 'r')

	items = json.load(fh)

	if not isinstance(items, list):
		sys.stderr.write('ERROR: Input must be a list\n')
		sys.exit(1)

	if not os.path.isdir('data'):
		os.makedirs('data')

	for item in items:
		url = item['URL']
		logging.info('URL=%s', url)
		try:
			item['Goose'] = json.loads(subprocess.check_output(['./hydrator', '-v', '-s', '127.0.0.1:8000', url]))
			with open('data/%s.json' % (item['ID'],), 'w') as fh:
				json.dump(item, fh)
		except subprocess.CalledProcessError as e:
			logging.error('Error for URL=%s: %s' % (url, e))

	sys.stdout.write(json.dumps(items))

if __name__ == '__main__':
	main(sys.argv)