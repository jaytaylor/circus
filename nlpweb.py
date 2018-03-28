#!/usr/bin/env python
# -*- coding: utf-8 -*-

from __future__ import print_function

import collections
import json
import spacy
import sys

from flask import Flask
from flask import jsonify
from flask import request

app = Flask(__name__)

nlp_instances = {}
default_instance = 'sm'

@app.route('/')
def index():
    return 'Hello world'

def clean_ents(ents):
    """
    Filter out undesirable entities and emit an ordered JSON-serializable
    structure.
    """
    by_freq = {}
    for ent in ents:
        if ent.label_ in (u'CARDINAL', u'ORDINAL', u'PERCENT', u'QUANTITY', u'DATE', u'MONEY'):
            continue
        if ent.text in by_freq:
            by_freq[ent.text][0] += 1
        else:
            by_freq[ent.text] = [1, ent.text, ent.label_, ent.root.tag_]

    final = list(map(
        # NB: args is of the form: 0->freq, 1->entity, 2->label, 3->part-of-speech.
        lambda args: { # freq, ent, label, pos: {
            'frequency': args[0], # freq,
            'entity': args[1], # ent,
            'label': args[2], # label,
            'pos': args[3], # pos,
        },
        list(
            collections.OrderedDict(
                sorted(by_freq.items(), key=lambda tup: tup[1][0], reverse=True)
            ).values()
        )
    ))
    return final


@app.route('/v1/named-entities', methods=['POST'])
def named_entities():
    global nlp_instances

    print(dir(request))

    instance = request.args.get('instance', default_instance)
    if instance in ('sm', 'md', 'lg') and not nlp_instances.get(instance):
        print('LOADING: en_core_web_%s' % (instance,))
        nlp_instances[instance] = spacy.load('en_core_web_%s' % (instance,))
    nlp_instance = nlp_instances[instance]

    if request.headers.get('Content-Type') == 'application/json':
        payload = json.loads(request.data)
        if not isinstance(payload, dict) or 'text' not in payload:
            raise Exception('missing "text" field in JSON payload')
        text = payload['text'].encode('utf-8')
    else:
        if not request.values:
            raise Exception('no data')

        text = max(list(request.values.keys()), key=len).encode('utf-8')

    doc = nlp_instance(text.decode('utf-8'))
    ents = doc.ents
    return jsonify(clean_ents(ents))

if __name__ == '__main__':
    host = '127.0.0.1'
    port = 8000
    if len(sys.argv) > 1:
        host, port = sys.argv[1].split(':', 1)
        port = int(port)
    app.run(host=host, port=port)

