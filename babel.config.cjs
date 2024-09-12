module.exports = {
  presets: [
    "@babel/preset-env",
    "@babel/preset-typescript",
    "@babel/preset-react",
  ],
  // required for easily mocking module exports.
  // see: https://stackoverflow.com/questions/67872622/jest-spyon-not-working-on-index-file-cannot-redefine-property
  assumptions: {
    constantReexports: true,
  },
  plugins: ["@babel/plugin-transform-modules-commonjs"],
};
