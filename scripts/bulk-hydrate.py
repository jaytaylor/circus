#!/usr/bin/env python
# -*- coding: utf-8 -*-

import argparse
import json
import logging
import os
import os.path
import subprocess
import sys


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def parse_flags(args):
    parser = argparse.ArgumentParser(description='Bulk hydrator for HN stories')
    parser.add_argument('--verbose', '-v', help='Enable verbose log output.', action='store_true')
    parser.add_argument('--quiet', '-q', help='Only log errors.', action='store_true')
    parser.add_argument('stories_json_file', help='Input file containing array of HN stories', nargs=1)
    parser.add_argument('output_dir', help='Output directory', nargs=1)

    flags = parser.parse_args(args)

    if flags.verbose and flags.quiet:
        raise Exception('Invalid parameter selection - at most 1 log level may be specified')

    if flags.verbose:
        logger.setLevel(logging.DEBUG)
    elif flags.quiet:
        logger.setLevel(logging.ERROR)

    return flags


def main(args):
    try:
        flags = parse_flags(args)
    except Exception as e:
        logger.error('%s', e)
        return 1

    try:
        if flags.stories_json_file[0] == '-':
            fh = sys.stdin
            items = json.load(fh)
        else:
            with open(flags.stories_json_file[0], 'r') as fh:
                items = json.load(fh)

        if not isinstance(items, list):
            logger.error('Input file must contain a JSON array')
            sys.exit(1)

        if not os.path.isdir(flags.output_dir[0]):
            os.makedirs(flags.output_dir[0])

        hydrator_bin = os.path.join(os.path.dirname(sys.argv[0]), 'goose-hydrator')

        subprocess.check_call(['go', 'build', '-o', hydrator_bin, '%s.go' % (hydrator_bin,)])

        for item in items:
            url = item['URL']
            logger.info('URL=%s', url)
            try:
                item['Goose'] = json.loads(subprocess.check_output([hydrator_bin, '-v', '-s', '127.0.0.1:8000', url]))
                with open('%s/%s.json' % (flags.output_dir[0], item['ID'],), 'w') as fh:
                    json.dump(item, fh)
            except subprocess.CalledProcessError as e:
                logger.error('Error for URL=%s: %s', url, e)

        logger.info('Processed %s items', len(items))

        #sys.stdout.write(json.dumps(items))

        # ppx@thing1:~/repos/test/quickstart$ rm -rf content/posts ; cd .. ; go run json2md.go foo && cp -a out quickstart/content/posts ; cd - ; rebuild^C
        # ppx@thing1:~/repos/test/quickstart$ type rebuild
        # rebuild is aliased to `rm -rf public/* && hugo && rm -rf /var/www/jaytaylor.com/public_html/hn && cp -a public /var/www/jaytaylor.com/public_html/hn'
    except:
        logging.exception('Main caught exception')
        return 1

    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))

