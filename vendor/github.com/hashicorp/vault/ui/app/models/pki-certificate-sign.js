import { copy } from 'ember-copy';
import { computed } from '@ember/object';
import DS from 'ember-data';
import Certificate from './pki-certificate';

const { attr } = DS;

export default Certificate.extend({
  signVerbatim: attr('boolean', {
    readOnly: true,
    defaultValue: false,
  }),

  csr: attr('string', {
    label: 'Certificate Signing Request (CSR)',
    editType: 'textarea',
  }),

  fieldGroups: computed('signVerbatim', function() {
    const options = { Options: ['altNames', 'ipSans', 'ttl', 'excludeCnFromSans', 'otherSans'] };
    const groups = [
      {
        default: ['csr', 'commonName', 'format', 'signVerbatim'],
      },
    ];
    if (this.get('signVerbatim') === false) {
      groups.push(options);
    }

    return this.fieldsToAttrs(copy(groups, true));
  }),
});
