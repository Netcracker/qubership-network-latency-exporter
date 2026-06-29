module.exports = {
  extends: ['@commitlint/config-conventional'],
  helpUrl: 'https://www.conventionalcommits.org/',
  ignores: [(msg) => /Signed-off-by: renovate\[bot]/m.test(msg)]
};
