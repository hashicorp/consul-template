import { computed } from '@ember/object';
import DS from 'ember-data';

import AuthConfig from '../auth-config';
import fieldToAttrs from 'vault/utils/field-to-attrs';

const { attr } = DS;

export default AuthConfig.extend({
  credentials: attr('string', {
    editType: 'file',
  }),

  googleCertsEndpoint: attr('string'),

  fieldGroups: computed(function() {
    const groups = [
      { default: ['credentials'] },
      {
        'Google Cloud Options': ['googleCertsEndpoint'],
      },
    ];
    return fieldToAttrs(this, groups);
  }),
});
