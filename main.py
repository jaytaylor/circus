#!/usr/bin/env python
# -*- coding: utf-8 -*-

from __future__ import print_function #, unicode_literals

import collections
import os
# import plac
import spacy
import sys

from pprint import pprint

def extract_currency_relations(nlp, text):
    #print(dir(doc))

    doc = nlp(text)
    ents0 = list(doc.ents)

    '''
    toks = []

    for tok in doc:
        print('%s %s | %s' % (tok.i, tok, tok.text.encode('ascii', 'ignore')))
        toks.append(tok)
        if tok.i == 1:
            print(dir(tok))
#            break

    # notice these 2 lines - if they're not here, standard NER
    # will be used and all scores will be 1.0
    with nlp.disable_pipes('ner'):
        doc = nlp(text)

    beams, somethingelse = nlp.entity.beam_parse([doc], beam_width=16, beam_density=0.0001)

    entity_scores = collections.defaultdict(float)
    for beam in beams:
        for score, ents in nlp.entity.moves.get_beam_parses(beam):
            for start, end, label in ents:
                entity_scores[(start, end, label)] += score

    print('--- Entities and scores (detected with beam search) ---')
    for key in entity_scores:
        start, end, label = key
        print('%d to %d: %s (%f) | %s' % (start, end - 1, label, entity_scores[key], toks[start].text))
    '''


    return ents0
#    # merge entities and noun chunks into one token
#    spans = list(doc.ents) + list(doc.noun_chunks)
#    for span in spans:
#        span.merge()
#
#    relations = []
#    for money in filter(lambda w: w.ent_type_ == 'MONEY', doc):
#        if money.dep_ in ('attr', 'dobj'):
#            subject = [w for w in money.head.lefts if w.dep_ == 'nsubj']
#            if subject:
#                subject = subject[0]
#                relations.append((subject, money))
#        elif money.dep_ == 'pobj' and money.head.dep_ == 'prep':
#            relations.append((money.head.head, money))
#    return relations

def process_and_clean_named_entities(ents):
    freq = {}
    print(dir(ents[0]))
    for ent in ents:
        if ent.label_ in (u'CARDINAL', u'ORDINAL', u'PERCENT', u'QUANTITY', u'DATE', u'MONEY'):
            continue
        # print('ent=%s v=%s' % (ent, freq.get(ent, None)))
        if ent.text in freq:
            freq[ent.text][1] += 1
        else:
            freq[ent.text] = [ent, 1, ent.label_]
    final = list(
        collections.OrderedDict(sorted(freq.items(), key=lambda tup: tup[1][1], reverse=True)).values()
    )
    return final

# @plac.annotations(model=('Model to load (needs parser and NER)', 'positional', None, str))
def main(args):
    # if len(args) < 2:
    #     sys.stderr.write('ERROR: Missing required parameter: [content-file]\n')
    #     os.exit(1)

    nlp = spacy.load('en_core_web_lg')
    #nlp = spacy.load('en_core_web_sm')

    if len(args) < 2 or args[1] == '-':
        fh = sys.stdin
    else:
        fh = open(args[1], 'r')

    try:
        text = fh.read().decode('utf-8')
    finally:
        fh.close()

    #print(dir(spacy))

    #doc = nlp(text.decode('utf-8'))

    nes = process_and_clean_named_entities(
        extract_currency_relations(nlp, text)
    )

    #nes = map(lambda x: '/%s/' % (x,), nes)

    pprint(nes)

if __name__ == '__main__':
    main(sys.argv)

