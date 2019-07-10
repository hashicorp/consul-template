import { inject as service } from '@ember/service';
import Component from '@ember/component';
import { task } from 'ember-concurrency';
import DS from 'ember-data';

const AuthConfigBase = Component.extend({
  tagName: '',
  model: null,

  flashMessages: service(),

  saveModel: task(function*() {
    try {
      yield this.get('model').save();
    } catch (err) {
      // AdapterErrors are handled by the error-message component
      // in the form
      if (err instanceof DS.AdapterError === false) {
        throw err;
      }
      return;
    }
    this.get('flashMessages').success('The configuration was saved successfully.');
  }),
});

AuthConfigBase.reopenClass({
  positionalParams: ['model'],
});

export default AuthConfigBase;
